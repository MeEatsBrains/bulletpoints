package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/meeatsbrains/bulletpoints"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type simulationResult struct {
	worstCase, bestCase int
	simulations         []int
	stagesSimulations   [][]int
}

func simulate(simulationCount int, optRuleAlterations *bulletpoints.RuleAlterations) (*simulationResult, error) {
	// first calculate best and worst case
	s := bulletpoints.NewSimu(bulletpoints.AllFail, optRuleAlterations)
	worstCase, _ := s.Simulate(1)
	s = bulletpoints.NewSimu(bulletpoints.AllSuccess, optRuleAlterations)
	bestCase, _ := s.Simulate(1)

	if worstCase < bestCase {
		return nil, fmt.Errorf("worst case = %d < best case = %d", worstCase, bestCase)
	}

	s = bulletpoints.NewSimu(bulletpoints.NormalMode, optRuleAlterations)
	values := make([]int, simulationCount)
	stagesSimulations := make([][]int, simulationCount)
	for i := range values {
		// us a different seed (> 0 to avoid time based seeding) for each run
		value, stages := s.Simulate(1000 + i)
		if value > worstCase {
			return nil, fmt.Errorf("iter %d: value = %d > worst case: %d", i, value, worstCase)
		}
		if value < bestCase {
			return nil, fmt.Errorf("iter %d: value = %d < best case: %d", i, value, bestCase)
		}

		values[i] = value
		stagesSimulations[i] = stages
	}

	return &simulationResult{
		bestCase:          bestCase,
		worstCase:         worstCase,
		simulations:       values,
		stagesSimulations: stagesSimulations,
	}, nil
}

func printStats(simulationRes *simulationResult) {
	sum := 0.0

	fmt.Printf("best case: %d bullets\nworst case: %d bullets\n", simulationRes.bestCase, simulationRes.worstCase)
	for _, value := range simulationRes.simulations {
		sum += float64(value)
	}
	if len(simulationRes.simulations) > 0 {
		fmt.Printf("average bullet count over %d simulations: %g bullets\n", len(simulationRes.simulations), sum/float64(len(simulationRes.simulations)))
	}
}

func generateHistogram(simulationRes *simulationResult, fileName string) (e error) {
	// panic recovery if required, transform it into an error
	defer func() {
		if r := recover(); r != nil {
			e = errors.Join(e, fmt.Errorf("panic: %v", r))
		}
	}()

	binCount := simulationRes.worstCase - simulationRes.bestCase + 1

	// move data into compatible type for plotting
	values := make(plotter.Values, len(simulationRes.simulations))
	for i, value := range simulationRes.simulations {
		values[i] = float64(value)
	}

	// generate a histogram plot
	p := plot.New()
	hist, err := plotter.NewHist(values, binCount)
	if err != nil {
		return fmt.Errorf("new hist: %w", err)
	}
	p.Add(hist)
	if err := p.Save(13*vg.Centimeter, 10*vg.Centimeter, fileName); err != nil {
		return fmt.Errorf("failed to save histogram: %w", err)
	}

	return nil
}

func generatePlotAndPrintBreakPoints(simulationRes *simulationResult, fileName string) (e error) {
	// panic recovery if required, transform it into an error
	defer func() {
		if r := recover(); r != nil {
			e = errors.Join(e, fmt.Errorf("panic: %v", r))
		}
	}()

	// count the number of simulations with a given bullet count
	buckets := make([]float64, simulationRes.worstCase+1)
	for _, value := range simulationRes.simulations {
		buckets[value]++
	}

	// accumulate the counts and make the entries in the buckets a relative percentage
	accumulated := make([]float64, len(buckets))
	accSum := 0.0
	simulationCount := len(simulationRes.simulations)

	breakPointInPerMille := 100
	for i := range buckets {
		if simulationCount > 0 {
			buckets[i] *= 100.0 / float64(simulationCount)
		}
		accSum += buckets[i]
		accumulated[i] = accSum
		for (breakPointInPerMille < 1000) && (accSum > 0.1*float64(breakPointInPerMille)) {
			fmt.Printf("chance of %.1f %% exceeded with %d bullets\n", float64(breakPointInPerMille)*0.1, i)
			if breakPointInPerMille < 900 {
				breakPointInPerMille += 100
			} else if breakPointInPerMille < 990 {
				breakPointInPerMille += 10
			} else {
				breakPointInPerMille += 1
			}
		}
	}

	// generate the plot compatible data structures with the points
	xyBucket := make(plotter.XYs, len(buckets))
	for i := range buckets {
		xyBucket[i].X = float64(i)
		xyBucket[i].Y = buckets[i]
	}
	xyAcc := make(plotter.XYs, len(accumulated))
	for i := range accumulated {
		xyAcc[i].X = float64(i)
		xyAcc[i].Y = accumulated[i]
	}

	// generate and save the plot
	p := plot.New()
	if err := plotutil.AddLines(p, "chance", xyBucket, "accumulated", xyAcc); err != nil {
		return fmt.Errorf("add lines: %w", err)
	}
	if err := p.Save(20*vg.Centimeter, 14*vg.Centimeter, fileName); err != nil {
		return fmt.Errorf("failed to save probability: %w", err)
	}

	return nil
}

func getReadableRuleAlterationOptions(ruleAlterationOptions *bulletpoints.RuleAlterations) string {
	if ruleAlterationOptions == nil {
		ruleAlterationOptions = &bulletpoints.RuleAlterations{}
	}

	ruleAlterationDescriptions := make([]string, 0)
	if ruleAlterationOptions.DontRemoveHitTargets {
		ruleAlterationDescriptions = append(ruleAlterationDescriptions, "hit targets are not removed and can be hit again")
	}
	if ruleAlterationOptions.NoPromisedWeightIncrease {
		ruleAlterationDescriptions = append(ruleAlterationDescriptions, "the promised increased chance to hit the major target is not honored")
	}

	if len(ruleAlterationDescriptions) == 0 {
		return "following the rules as stated"
	}

	return fmt.Sprintf("alterations included in the ruleset: %s", strings.Join(ruleAlterationDescriptions, ", "))
}

func getRuleAlterationSuffix(ruleAlterationOptions *bulletpoints.RuleAlterations) string {
	if ruleAlterationOptions == nil {
		return ""
	}

	parts := make([]string, 0)

	if ruleAlterationOptions.DontRemoveHitTargets {
		parts = append(parts, "targets_not_removed")
	}
	if ruleAlterationOptions.NoPromisedWeightIncrease {
		parts = append(parts, "weight_not_increased_as_promised")
	}

	if len(parts) == 0 {
		return ""
	}

	return "_" + strings.Join(parts, "_and_")
}

func main() {
	const SIMULATION_COUNT = 100000
	const FMT_HISTOGRAM_FILE_NAME = "histo%s.png"
	const FMT_PROBABILITY_FILE_NAME = "probability%s.png"
	const FMT_RUN_BOX_PLOT_FILE_NAME = "run_comparison%s.png"
	const RECORDED_RUN_FILE_NAME = "recorded_run.csv"

	for _, ruleAlteration1 := range []bool{false, true} {
		for _, ruleAlteration2 := range []bool{false, true} {
			ruleAlterationOptions := bulletpoints.RuleAlterations{
				NoPromisedWeightIncrease: ruleAlteration1,
				DontRemoveHitTargets:     ruleAlteration2,
			}

			fileSuffix := getRuleAlterationSuffix(&ruleAlterationOptions)
			histogramFn := fmt.Sprintf(FMT_HISTOGRAM_FILE_NAME, fileSuffix)
			probabilityFn := fmt.Sprintf(FMT_PROBABILITY_FILE_NAME, fileSuffix)
			boxComparisonFn := fmt.Sprintf(FMT_RUN_BOX_PLOT_FILE_NAME, fileSuffix)

			fmt.Printf("starting simulation with %d runs (%s)\n", SIMULATION_COUNT, getReadableRuleAlterationOptions(&ruleAlterationOptions))
			simulationRes, err := simulate(SIMULATION_COUNT, &ruleAlterationOptions)
			if err != nil {
				fmt.Printf("simulation failed: %v", err)
				os.Exit(1)
			}

			printStats(simulationRes)
			fmt.Printf("now generating %s\n", histogramFn)
			err = generateHistogram(simulationRes, histogramFn)
			if err != nil {
				fmt.Printf("WARNING: failed to generate histogram: %v", err)
			}

			fmt.Printf("now generating %s\n", probabilityFn)
			err = generatePlotAndPrintBreakPoints(simulationRes, probabilityFn)
			if err != nil {
				fmt.Printf("WARNING: failed to generate probability plot: %v", err)
			}

			fmt.Printf("now loading %s\n", RECORDED_RUN_FILE_NAME)
			runData, err := loadRunData(RECORDED_RUN_FILE_NAME)
			if err != nil {
				fmt.Printf("WARNING: failed to load recorded run data: %v", err)
			} else {
				fmt.Printf("now generating %s\n", boxComparisonFn)
				err = generateRunDataBoxPlot(simulationRes, runData, boxComparisonFn)
				if err != nil {
					fmt.Printf("WARNING: failed to generate run data plot: %v", err)
				}
			}

			fmt.Println("=====================================")
			fmt.Println("")
		}
	}
}

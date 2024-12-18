package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/meeatsbrains/bulletpoints"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type simulationResult struct {
	worstCase, bestCase int
	simulations         []int
}

func simulate(simulationCount int) (*simulationResult, error) {
	// first calculate best and worst case
	s := bulletpoints.NewSimu(bulletpoints.AllFail)
	worstCase := s.Simulate(1)
	s = bulletpoints.NewSimu(bulletpoints.AllSuccess)
	bestCase := s.Simulate(1)

	if worstCase < bestCase {
		return nil, fmt.Errorf("worst case = %d < best case = %d", worstCase, bestCase)
	}

	s = bulletpoints.NewSimu(bulletpoints.NormalMode)
	values := make([]int, simulationCount)
	for i := range values {
		// us a different seed (> 0 to avoid time based seeding) for each run
		value := s.Simulate(1000 + i)
		if value > worstCase {
			return nil, fmt.Errorf("iter %d: value = %d > worst case: %d", i, value, worstCase)
		}
		if value < bestCase {
			return nil, fmt.Errorf("iter %d: value = %d < best case: %d", i, value, bestCase)
		}

		values[i] = value
	}

	return &simulationResult{
		bestCase:    bestCase,
		worstCase:   worstCase,
		simulations: values,
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

	const BIN_COUNTS = 100

	// move data into compatible type for plotting
	values := make(plotter.Values, len(simulationRes.simulations))
	for i, value := range simulationRes.simulations {
		values[i] = float64(value)
	}

	// generate a histogram plot
	p := plot.New()
	hist, err := plotter.NewHist(values, BIN_COUNTS)
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

	breakPoint := 10.0
	for i := range buckets {
		if simulationCount > 0 {
			buckets[i] *= 100.0 / float64(simulationCount)
		}
		accSum += buckets[i]
		accumulated[i] = accSum
		for (breakPoint < 100.0) && (accSum > breakPoint) {
			fmt.Printf("chance of %.0f %% exceeded with %d bullets\n", breakPoint, i)
			if breakPoint < 90 {
				breakPoint += 10.0
			} else {
				breakPoint += 1.0
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

func main() {
	const SIMULATION_COUNT = 100000
	const HISTOGRAM_FILE_NAME = "histo.png"
	const PROBABILITY_FILE_NAME = "probability.png"

	fmt.Printf("starting simulation with %d runs\n", SIMULATION_COUNT)
	simulationRes, err := simulate(SIMULATION_COUNT)
	if err != nil {
		fmt.Printf("simulation failed: %v", err)
		os.Exit(1)
	}

	printStats(simulationRes)
	fmt.Printf("now generating %s\n", HISTOGRAM_FILE_NAME)
	err = generateHistogram(simulationRes, HISTOGRAM_FILE_NAME)
	if err != nil {
		fmt.Printf("WARNING: failed to generate histogram: %v", err)
	}

	fmt.Printf("now generating %s\n", PROBABILITY_FILE_NAME)
	err = generatePlotAndPrintBreakPoints(simulationRes, PROBABILITY_FILE_NAME)
	if err != nil {
		fmt.Printf("WARNING: failed to generate probability plot: %v", err)
	}
}

package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func loadRunData(fileName string) ([]int, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	csvReader := csv.NewReader(file)

	data, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse csv data from %s: %w", fileName, err)
	}

	if len(data) != 101 {
		return nil, fmt.Errorf("unexpected number of lines in %s, found %d", fileName, len(data))
	}
	if len(data[0]) != 2 {
		return nil, fmt.Errorf("unexpected number of columns in %s, found %d", fileName, len(data[0]))
	}

	res := make([]int, 100)
	for lineNo, line := range data[1:] {
		count, err := strconv.Atoi(line[1])
		if err != nil {
			return nil, fmt.Errorf("line #%d in %s is malformed", 1+lineNo, fileName)
		}
		res[lineNo] = count
	}

	return res, nil
}

func generateRunDataBoxPlot(simRes *simulationResult, runData []int, fileName string) error {
	if simRes == nil {
		return errors.New("missing parameter: simulation result")
	}

	const stagesRepeatAfter = 5
	const runLimitForPlot = 1000
	stagesForPlot := simRes.stagesSimulations
	if runLimitForPlot < len(simRes.stagesSimulations) {
		stagesForPlot = simRes.stagesSimulations[:runLimitForPlot]
	}

	realDataPerStage := len(runData) / stagesRepeatAfter
	pointValues := make(plotter.XYZs, 0)
	boxPlotters := make([]plot.Plotter, stagesRepeatAfter)
	for col := range stagesRepeatAfter {
		boxValues := make(plotter.Values, 0)
		for _, data := range stagesForPlot {
			for idx := col; idx < len(data); idx += stagesRepeatAfter {
				boxValues = append(boxValues, float64(data[idx]))
			}
		}

		var err error
		boxPlotter, err := plotter.NewBoxPlot(vg.Points(20), float64(col), boxValues)
		if err != nil {
			return fmt.Errorf("failed to add boxplot #%d: %w", col, err)
		}
		boxPlotter.MedianStyle.Color = plotutil.Color(0)
		boxPlotter.MedianStyle.Width = 2.0 * boxPlotter.MedianStyle.Width
		boxPlotters[col] = boxPlotter

		dataPointsSoFar := 0.0
		for idx := col; idx < len(runData); idx += stagesRepeatAfter {
			pointValues = append(pointValues, plotter.XYZ{
				X: float64(col) + 0.2*dataPointsSoFar/float64(realDataPerStage) - 0.1,
				Y: float64(runData[idx]),
			})
			dataPointsSoFar++
		}
	}

	p := plot.New()
	scatter, err := plotter.NewScatter(pointValues)
	if err != nil {
		return fmt.Errorf("failed to create scatter plot: %w", err)
	}
	scatter.GlyphStyle.Color = plotutil.Color(1)

	p.Add(scatter)
	p.Add(boxPlotters...)
	if err := p.Save(13*vg.Centimeter, 10*vg.Centimeter, fileName); err != nil {
		return fmt.Errorf("failed to save run data plot: %w", err)
	}

	return nil
}

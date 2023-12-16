package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/barchart"
)

var max int = 1

func displayChart(plottingFunction func(ctx context.Context, bc *barchart.BarChart)) {
	t, err := tcell.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()

	ctx, cancel := context.WithCancel(context.Background())
	bc, err := barchart.New(
		barchart.ShowValues(),
		barchart.BarWidth(8),
	)
	if err != nil {
		panic(err)
	}
	go plottingFunction(ctx, bc)

	c, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle("PRESS Q TO QUIT"),
		container.PlaceWidget(bc),
	)
	if err != nil {
		panic(err)
	}

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' {
			cancel()
		}
	}

	if err := termdash.Run(ctx, t, c, termdash.KeyboardSubscriber(quitter)); err != nil {
		panic(err)
	}
}

func procsTotalTimePlotter(ctx context.Context, bc *barchart.BarChart) {

	for {

		var __processesWithRuntime *treebidimap.Map
		__processesWithRuntime = <-processesWithRuntime
		var keys []string
		var values []int
		var barColors []cell.Color

		itemsProcessed := 0

		processElement := func(runtime_o runtime_t) {

			pid := runtime_o.pid

			logger.Info(fmt.Sprintf("Processing item: %d, val: %v", pid, runtime_o))

			runtime := int(runtime_o.Time)

			barColors = append(barColors, getColor(runtime))

			values = append(values, runtime)
			var __key string

			if trackPID > 0 {
				__key = fmt.Sprintf("%d", runtime_o.cpuID)
			} else {
				if showPIDs {
					__key = fmt.Sprintf("%d", pid)
				} else {
					__key = strings.Replace(runtime_o.comm, "\x00", "", -1)
				}
			}
			keys = append(keys, __key)

			if int(runtime_o.Time) > max {
				max = int(runtime_o.Time)
			}
			itemsProcessed++
		}

		processed := 0
		for _, runtime_o := range __processesWithRuntime.Values() {
			if processed > showItems {
				break
			}
			processElement(runtime_o.(runtime_t))
			processed += 1
		}

		logger.Info(fmt.Sprintf("keys : %v, vals: %v", keys, values))

		labelOptions := barchart.Labels(keys)
		colorOptions := barchart.BarColors(barColors)
		if err := bc.Values(values, max, labelOptions, colorOptions); err != nil {
			panic(err)
		}
		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}

func getColor(value int) cell.Color {
	/*ballpark numbers, @TODO re-adjust colors based on overall load*/
	total := int(refreshInterval)
	if value <= total/5 {
		return cell.ColorGreen
	} else if value > total/5 && value <= total/20 {
		return cell.ColorOlive
	} else {
		return cell.ColorRed
	}
}

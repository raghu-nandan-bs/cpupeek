package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/barchart"
)

var max int = 1 // 1 * 1000 * 1000 * 1000 // max value for the barchart

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
	const maxItems = 20

	for {
		var __processesWithRuntime *treebidimap.Map

		__processesWithRuntime = <-processesWithRuntime
		var keys []string
		var values []int

		procData := __processesWithRuntime.Values()

		itemsProcessed := 0
		for i := range procData {
			if trackPID > 0 {
				values = append(values, int(procData[len(procData)-i-1].(CPUTimeAggrByCPU).Time))
				key := fmt.Sprintf("%d", procData[len(procData)-i-1].(CPUTimeAggrByCPU).cpuID)

				keys = append(keys, key)
				if int(procData[len(procData)-i-1].(CPUTimeAggrByCPU).Time) > max {
					max = int(procData[len(procData)-i-1].(CPUTimeAggrByCPU).Time)
				}
				itemsProcessed++
				if itemsProcessed >= maxItems {
					break
				}
			} else {
				values = append(values, int(procData[len(procData)-i-1].(CPUTimeAggrByProcess).Time))
				key := strings.Replace(procData[len(procData)-i-1].(CPUTimeAggrByProcess).ProcessName, "\x00", "", -1)
				keys = append(keys, key)
				if int(procData[len(procData)-i-1].(CPUTimeAggrByProcess).Time) > max {
					max = int(procData[len(procData)-i-1].(CPUTimeAggrByProcess).Time)
				}
				itemsProcessed++
				if itemsProcessed >= maxItems {
					break
				}
			}
		}
		options := barchart.Labels(keys)

		if err := bc.Values(values, max, options); err != nil {
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

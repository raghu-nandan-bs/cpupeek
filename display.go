package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/barchart"
)

var max int = 1 // 1 * 1000 * 1000 * 1000 // max value for the barchart
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 -]+`)

func clearString(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, "")
}

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
		var key string
		var keys []string
		var values []int
		procData := __processesWithRuntime.Values()

		itemsProcessed := 0
		for i := range procData {

			idx := len(procData) - i - 1
			runtime := int(procData[idx].(runtime_t).Time)

			values = append(values, runtime)

			if trackPID > 0 {
				key = fmt.Sprintf("%d", procData[idx].(runtime_t).cpuID)
				logger.Debug("key: " + key)
			} else {
				key = clearString(procData[idx].(runtime_t).comm)
			}
			key = key
			keys = append(keys, key)
			if int(procData[idx].(runtime_t).Time) > max {
				max = int(procData[idx].(runtime_t).Time)
			}
			itemsProcessed++
			if itemsProcessed >= showItems {
				break
			}
		}
		labelOptions := barchart.Labels(keys)

		if err := bc.Values(values, max, labelOptions); err != nil {
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

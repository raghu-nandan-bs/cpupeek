package main

import (
	"C"
	_ "embed"
	"fmt"

	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/aquasecurity/libbpfgo/helpers"
	"github.com/zerodha/logf"
)
import (
	"context"
	"flag"
	"github.com/aquasecurity/libbpfgo"
	"io"
	"os"
	"strings"
	"time"
)

//go:embed cpupeek.bpf.o
var bpfBin []byte
var bpfName = "cpupeek.bpf.o"

var (
	ctx    context.Context
	cancel context.CancelFunc
)

var numCpus int = 0
var trackPID int64 = -1
var trackCPU int64 = -1
var trueScale bool = false
var showItems int = 20
var showPIDs bool = false
var refreshInterval time.Duration = 1 * time.Second

var logFile string
var logger logf.Logger

func init() {
	flag.Int64Var(&trackPID, "pid", -1, "pid to track")
	flag.BoolVar(&showPIDs, "show-pids", false, "display process id instead of their names.")
	flag.Int64Var(&trackCPU, "cpu", -1, "cpu to track")
	flag.BoolVar(&trueScale, "true-scale", false, "scale the barchart to 1s (y axis)")
	flag.StringVar(&logFile, "log", "cpupeek.log", "log file to write to")
	flag.IntVar(&showItems, "show-items", 15, "number of items to show in the barchart")
	flag.DurationVar(&refreshInterval, "interval", 1*time.Second, "how often should the screen refresh?")

	flag.Parse()

	// open log file in append mode
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	// get io.Writer from file
	w := io.Writer(file)
	logger = logf.New(logf.Opts{
		Writer:               w,
		EnableColor:          false,
		Level:                logf.DebugLevel,
		CallerSkipFrameCount: 3,
		EnableCaller:         true,
		TimestampFormat:      time.RFC3339Nano,
	})

	// if trueScale is enabled, max should be 1s
	if trueScale {
		max = 1000000000
	}

	bpfLogFile := strings.Replace(logFile, ".log", ".bpf.log", -1)
	bpfLogFileStream, err := os.OpenFile(bpfLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	os.Stdout = bpfLogFileStream
	if err != nil {
		panic(err)
	}

}

func main() {
	logger.Info("Starting cpupeek")
	numCpus = cpuCount()

	module, err := bpf.NewModuleFromBuffer(bpfBin, bpfName)
	if err != nil {
		panic(err)
	}
	defer module.Close()

	err = module.InitGlobalVariable("pid", trackPID)
	if err != nil {
		panic(err)
	}
	err = module.InitGlobalVariable("cpu", trackCPU)
	if err != nil {
		panic(err)
	}
	err = module.BPFLoadObject()
	if err != nil {
		panic(err)
	}
	prog, err := module.GetProgram("trace_sched_stat_runtime")
	if err != nil {
		panic(err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	_, err = prog.AttachGeneric()
	if err != nil {
		panic(fmt.Sprintf("failed to attach program (%s): %v", prog.Name(), err))
	}

	go helpers.TracePipeListen()

	run(module)

}

func run(module *libbpfgo.Module) {
	runtime_arr, err := module.GetMap("runtime_arr")

	if err != nil {
		panic(err)
	}
	timer := time.NewTicker(refreshInterval)
	go func() {
		for {
			select {
			case <-timer.C:
				processData(runtime_arr)
			case <-ctx.Done():
				return
			}
		}
	}()

	displayChart(procsTotalTimePlotter)

	return
}

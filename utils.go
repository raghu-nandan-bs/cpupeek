package main

import (
	"encoding/binary"
	"unsafe"

	"github.com/aquasecurity/libbpfgo"
	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/emirpasic/gods/utils"
	"github.com/prometheus/procfs"
)

// Generic helpers
func cpuCount() int {
	fs, err := procfs.NewFS("/proc")
	cpus, err := fs.CPUInfo()
	if err != nil {
		panic(err)
	}
	return len(cpus)
}

var processesWithRuntime = make(chan *treebidimap.Map, 2)

type CPUTime struct {
	CPU         uint16
	Time        uint64
	ProcessName string
	pid         int
}

func cpuTimeByRuntime(this, other interface{}) int {
	__this := this.(CPUTime)
	__other := other.(CPUTime)
	if __this.Time < __other.Time {
		return -1
	} else if __this.Time > __other.Time {
		return 1
	}
	return 0
}

type CPUTimeAggrByProcess struct {
	ProcessName string
	Time        uint64
}

type CPUTimeAggrByCPU struct {
	cpuID uint16
	Time  uint64
}

func sortCpuTimeAggrByProcess(this, other interface{}) int {
	__this := this.(CPUTimeAggrByProcess)
	__other := other.(CPUTimeAggrByProcess)
	if __this.Time < __other.Time {
		return -1
	} else if __this.Time > __other.Time {
		return 1
	}
	return 0
}

func sortCpuTimeAggrByCPU(this, other interface{}) int {
	__this := this.(CPUTimeAggrByCPU)
	__other := other.(CPUTimeAggrByCPU)
	if __this.Time < __other.Time {
		return -1
	} else if __this.Time > __other.Time {
		return 1
	}
	return 0
}

func processesByCPUFromBPFMap(runtimeMap *libbpfgo.BPFMap) {
	var procs *treebidimap.Map = treebidimap.NewWith(utils.UInt16Comparator, sortCpuTimeAggrByCPU)
	sizeOfComm := 16
	sizeOfCpuID := 8
	sizeOfTotalRuntime := 8

	iter := runtimeMap.Iterator()
	for iter.Next() {
		var pid uint32
		pid = binary.LittleEndian.Uint32(iter.Key())

		rawValue := make([]byte, 32*numCpus)
		err := runtimeMap.GetValueReadInto(unsafe.Pointer(&pid), &rawValue)

		if err != nil {
			logger.Error("failed to get comm value:" + err.Error())
		}

		for i := 0; i < numCpus*32; i = i + 32 {
			totalRuntime := binary.LittleEndian.Uint64(rawValue[i+sizeOfComm+sizeOfCpuID : i+sizeOfComm+sizeOfCpuID+sizeOfTotalRuntime])

			__cpuID := binary.LittleEndian.Uint16(rawValue[i+sizeOfComm : i+sizeOfComm+sizeOfCpuID])
			if totalRuntime > 0 {
				val, found := procs.Get(__cpuID)
				if found {
					// update the value
					totalRuntime += val.(CPUTimeAggrByCPU).Time
				}

				procs.Put(__cpuID, CPUTimeAggrByCPU{cpuID: __cpuID, Time: totalRuntime})
			}
		}
		runtimeMap.DeleteKey(unsafe.Pointer(&pid))
	}
	processesWithRuntime <- procs

}

func processesByRuntimeFromBPFMap(runtimeMap *libbpfgo.BPFMap) {
	var procs *treebidimap.Map = treebidimap.NewWith(utils.StringComparator, sortCpuTimeAggrByProcess)
	sizeOfComm := 16
	sizeOfCpuID := 8
	sizeOfTotalRuntime := 8

	iter := runtimeMap.Iterator()
	for iter.Next() {
		var pid uint32
		pid = binary.LittleEndian.Uint32(iter.Key())

		rawValue := make([]byte, 32*numCpus)
		err := runtimeMap.GetValueReadInto(unsafe.Pointer(&pid), &rawValue)

		if err != nil {
			logger.Error("failed to get comm value:" + err.Error())
		}
		for i := 0; i < numCpus*32; i = i + 32 {
			comm := string(rawValue[i : i+sizeOfComm])
			totalRuntime := binary.LittleEndian.Uint64(rawValue[i+sizeOfComm+sizeOfCpuID : i+sizeOfComm+sizeOfCpuID+sizeOfTotalRuntime])

			if totalRuntime > 0 {
				val, found := procs.Get(comm)
				if found {
					totalRuntime += val.(CPUTimeAggrByProcess).Time
				}
				procs.Put(comm, CPUTimeAggrByProcess{ProcessName: comm, Time: totalRuntime})
			}
		}
		runtimeMap.DeleteKey(unsafe.Pointer(&pid))
	}
	processesWithRuntime <- procs

}

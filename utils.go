package main

import (
	"encoding/binary"
	"unsafe"

	"github.com/aquasecurity/libbpfgo"
	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/emirpasic/gods/utils"
	"github.com/prometheus/procfs"
)

const (
	commBitLength    = 16
	cpuIDBitLength   = 8
	runtimeBitLength = 8
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

type runtime_t struct {
	comm  string
	cpuID uint16
	Time  uint64
}

func sortRuntime(this, other interface{}) int {
	__this := this.(runtime_t)
	__other := other.(runtime_t)
	if __this.Time < __other.Time {
		return -1
	} else if __this.Time > __other.Time {
		return 1
	}
	return 0
}

func processData(processRuntimeMap *libbpfgo.BPFMap) {

	var procs *treebidimap.Map = newBTreeMap()
	dataStream := processRuntimeMap.Iterator()

	for dataStream.Next() {
		var pid uint32
		pid = binary.LittleEndian.Uint32(dataStream.Key())
		rawValue := make([]byte, 32*numCpus)

		err := processRuntimeMap.GetValueReadInto(
			unsafe.Pointer(&pid),
			&rawValue,
		)
		if err != nil {
			logger.Error("failed to runtime data:" + err.Error())
		}

		for i := 0; i < numCpus*32; i = i + 32 {
			runtimeInfo := extract(rawValue[i : i+32])
			if trackPID > 0 {
				storeByCPU(procs, runtimeInfo)
			} else {
				storeByProcess(procs, runtimeInfo)
			}
		}
		processRuntimeMap.DeleteKey(unsafe.Pointer(&pid))
	}
	processesWithRuntime <- procs
}

func extract(rawValue []byte) runtime_t {

	runtime := binary.LittleEndian.Uint64(rawValue[commBitLength+cpuIDBitLength : commBitLength+cpuIDBitLength+runtimeBitLength])
	if runtime == 0 {
		return runtime_t{
			Time: 0,
		}
	}
	cpuId := binary.LittleEndian.Uint16(rawValue[commBitLength : commBitLength+cpuIDBitLength])
	comm := string(rawValue[:commBitLength])
	return runtime_t{
		comm:  comm,
		cpuID: cpuId,
		Time:  runtime,
	}
}

func storeByProcess(
	mapStore *treebidimap.Map,
	runtimeInfo runtime_t,
) {
	currentValue, exists := mapStore.Get(runtimeInfo.comm)
	if !exists {
		mapStore.Put(runtimeInfo.comm, runtimeInfo)
		return
	}
	updatedValue := currentValue.(runtime_t)
	updatedValue.Time += runtimeInfo.Time
	mapStore.Put(runtimeInfo.comm, updatedValue)
}

func storeByCPU(
	mapStore *treebidimap.Map,
	runtimeInfo runtime_t,
) {
	currentValue, exists := mapStore.Get(runtimeInfo.cpuID)
	if !exists {
		mapStore.Put(runtimeInfo.cpuID, runtimeInfo)
		return
	}
	updatedValue := currentValue.(runtime_t)
	updatedValue.Time += runtimeInfo.Time
	mapStore.Put(runtimeInfo.cpuID, updatedValue)
}

func newBTreeMap() *treebidimap.Map {
	if trackPID > 0 {
		return treebidimap.NewWith(utils.UInt16Comparator, sortRuntime)
	}
	return treebidimap.NewWith(utils.StringComparator, sortRuntime)
}

package treadmarks

import "sync"

type DataStructureInterface interface {
	LockDatastructure()
	UnlockDatastructure()
	PageArrayInterface1
	ProcArrayInterface1
}

type DataStructure struct {
	*PageArray1
	*ProcArray1
	procId *byte
	*sync.Mutex
}

type Diff struct {
	Diffs []Pair
}

func (d *DataStructure) LockDatastructure() {
	d.Mutex.Lock()
}

func (d *DataStructure) UnlockDatastructure() {
	d.Mutex.Unlock()
}

func CreateDiff(original, new []byte) Diff {
	res := make([]Pair, 0)
	for i, data := range original {
		if new[i] != data {
			res = append(res, Pair{i, new[i]})
		}
	}
	return Diff{res}
}

type Pair struct {
	Car, Cdr interface{}
}

func NewDataStructure(procId *byte, nrProcs int) *DataStructure {
	ds := new(DataStructure)
	ds.procId = procId
	ds.ProcArray1 = NewProcArray(nrProcs)
	ds.PageArray1 = NewPageArray1(procId, nrProcs)
	ds.Mutex = new(sync.Mutex)
	return ds
}

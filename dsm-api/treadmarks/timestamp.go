package treadmarks



type Timestamp []int32

func NewTimestamp(nrProcs uint8) Timestamp {
	return make([]int32, nrProcs)
}

func (t Timestamp) increment(procId uint8) {
	t[procId]++
}

func (t Timestamp) covers(o Timestamp) bool {
	if len(t) != len(o) {
		panic("Compared timestamps of different length.")
	}
	for i := 0; i < len(t); i++{
		if t[i]<o[i] {
			return false
		}
	}
	return true
}

func (t Timestamp) merge(o Timestamp) Timestamp {
	newTs := NewTimestamp(uint8(len(t)))
	for i := range t {
		if t[i] > o[i] {
			newTs[i] = t[i]
		} else {
			newTs[i] = o[i]
		}
	}
	return newTs
}

func (t Timestamp) min(o Timestamp) Timestamp {
	newTs := NewTimestamp(uint8(len(t)))
	for i := range t {
		if t[i] < o[i] {
			newTs[i] = t[i]
		} else {
			newTs[i] = o[i]
		}
	}
	return newTs
}


func (t Timestamp) loadFromData(data []byte){
	for i := range t{
		t[i] = BytesToInt32(data[i*4:i*4+4])
	}
}

func (t Timestamp) saveToData() []byte {
	data := make([]byte, 4*len(t))
	for i := range t{
		copy(data[i*4:i*4+3], Int32ToBytes(t[i]))
	}
	return data
}
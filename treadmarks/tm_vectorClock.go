package treadmarks

type Vectorclock struct {
	Value []uint
}

func (v *Vectorclock) Compare(other *Vectorclock) int {
	isBefore := 0
	isAfter := 0
	for i := 0; i < len(v.Value); i++ {
		if v.Value[i] < other.Value[i] {
			isBefore = -1
		} else if v.Value[i] > other.Value[i] {
			isAfter = 1
		}
	}
	return isAfter + isBefore
}

func NewVectorclock(nodes int) *Vectorclock {
	v := new(Vectorclock)
	v.Value = make([]uint, nodes)
	return v
}

func (v *Vectorclock) Merge(o Vectorclock) *Vectorclock {
	nv := new(Vectorclock)
	nv.Value = make([]uint, len(v.Value))
	for i := range nv.Value {
		nv.Value[i] = Max(v.Value[i], o.Value[i])
	}
	return nv
}

func (v *Vectorclock) Increment(id byte) {
	v.Value[int(id)-1] = v.Value[int(id)-1] + 1
}

//Returns true of v is before o causally.
func (v *Vectorclock) IsBefore(o *Vectorclock) bool {
	return v.Compare(o) == -1
}

//Returns true if v is after o causally.
func (v *Vectorclock) IsAfter(o *Vectorclock) bool {
	return v.Compare(o) == 1
}

func (v *Vectorclock) GetTick(id byte) uint {
	return v.Value[int(id)-1]
}

func (v *Vectorclock) SetTick(id byte, tick uint) {
	v.Value[int(id)-1] = tick
}

//Returns true if every entry in v is equal to the matching entry in o
func (v *Vectorclock) Equals(o *Vectorclock) bool {
	if len(v.Value) == len(o.Value) {
		for i := range v.Value {
			if v.Value[i] != o.Value[i] {
				return false
			}
		}
		return true
	}
	return false
}

// Here is some utility stuff
func Max(x, y uint) uint {
	if x > y {
		return x
	}
	return y
}

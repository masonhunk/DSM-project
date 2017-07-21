package treadmarks


type Vectorclock struct {
	value []uint
}

func (v *Vectorclock) Compare(other Vectorclock) int {
	isBefore := 0
	isAfter := 0
	for i := 0; i < len(v.value); i++ {
		if v.value[i] < other.value[i] {
			isBefore = -1
		} else if v.value[i] > other.value[i] {
			isAfter = 1
		}
	}
	return isAfter + isBefore
}

func NewVectorclock(nodes int) *Vectorclock {
	v := new(Vectorclock)
	v.value = make([]uint, nodes)
	return v
}

func (v *Vectorclock) Merge(o Vectorclock) *Vectorclock{
	nv := new(Vectorclock)
	nv.value = make([]uint, len(v.value))
	for i := range nv.value{
		nv.value[i] = Max(v.value[i], o.value[i])
	}
	return nv
}

func (v *Vectorclock) Increment(id byte){
	v.value[id] = v.value[id]+1
}

func (v *Vectorclock) IsBefore(o Vectorclock) bool{
	return v.Compare(o) == -1
}

func (v *Vectorclock) IsAfter(o Vectorclock) bool{
	return v.Compare(o) == 1
}

func (v *Vectorclock) GetTick(id byte) uint{
	return v.value[id]
}

func (v *Vectorclock) SetTick(id byte, tick uint){
	v.value[id] = tick
}

// Here is some utility stuff
func Max(x, y uint) uint {
	if x > y {
		return x
	}
	return y
}
package treadmarks


type Vectorclock struct {
	value []uint
}

func (v *Vectorclock) Compare(other *Vectorclock) int {
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

func (v *Vectorclock) Increment(id int){
	v.value[id] = v.value[id]+1
}

func (v *Vectorclock) GetTick(id int) uint{
	return v.value[id]
}

func (v *Vectorclock) SetTick(id int, tick uint){
	v.value[id] = tick
}
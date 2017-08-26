package treadmarks

type LockAcquireRequest struct{
	From      uint8
	LockId    uint8
	Timestamp Timestamp
}

type LockAcquireResponse struct{
	LockId    uint8
	Intervals []IntervalRecord
}

type BarrierRequest struct{
	From      uint8
	BarrierId uint8
	Intervals []IntervalRecord
}

type BarrierResponse struct{
	Intervals []IntervalRecord
}

type DiffRequest struct {
	From   uint8
	PageNr int16
	FromTs Timestamp
	ToTs   Timestamp
}

type DiffResponse struct {
	PageNr       int16
	Writenotices []WritenoticeRecord
}

type CopyRequest struct {
	From uint8
	PageNr int16
}

type CopyResponse struct {
	PageNr int16
	Data []byte
}



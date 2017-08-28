package treadmarks


type LockAcquireRequest struct{
	From      uint8 `xdropaque:"false"`
	LockId    uint8 `xdropaque:"false"`
	Timestamp Timestamp
}

type LockAcquireResponse struct{
	LockId    uint8 `xdropaque:"false"`
	Timestamp Timestamp
	Intervals []IntervalRecord
}

type BarrierRequest struct{
	From      uint8 `xdropaque:"false"`
	BarrierId uint8 `xdropaque:"false"`
	Timestamp Timestamp
	Intervals []IntervalRecord
}

type BarrierResponse struct{
	Intervals []IntervalRecord
	Timestamp Timestamp
}

type DiffRequest struct {
	From   uint8 `xdropaque:"false"`
	to     uint8
	PageNr int16
	First  Timestamp
	Last   Timestamp
}

type DiffResponse struct {
	PageNr       int16
	Writenotices []WritenoticeRecord
}

type CopyRequest struct {
	From uint8 `xdropaque:"false"`
	PageNr int16 `xdropaque:"false"`
	Timestamp Timestamp
}

type CopyResponse struct {
	PageNr int16 `xdropaque:"false"`
	Data []byte
}
/*
func (l LockAcquireRequest) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(l.From)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = enc.Encode(l.LockId)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = enc.Encode(l.Timestamp)
	if err != nil {
		fmt.Println(err.Error())
	}
	return buf.Bytes(), err
}

func (l *LockAcquireRequest) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&l.From)
	if err != nil{
		fmt.Println(err.Error())
		return err
	}
	err = dec.Decode(&l.LockId)
	if err != nil{
		fmt.Println(err.Error())
		return err
	}
	err = dec.Decode(&l.Timestamp)
	if err != nil{
		fmt.Println(err.Error())
		return err
	}
	fmt.Println("decoded LockAcquireRequest", l)
	return nil
}

func (t DiffRequest) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(t.From)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = enc.Encode(t.To)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = enc.Encode(t.PageNr)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = enc.Encode(t.First)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = enc.Encode(t.Last)
	if err != nil {
		fmt.Println(err.Error())
	}
	return buf.Bytes(), err
}

func (t *DiffRequest) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&t.From)
	if err != nil{
		fmt.Println(err.Error())
		return err
	}
	err = dec.Decode(&t.To)
	if err != nil{
		fmt.Println(err.Error())
		return err
	}

	err = dec.Decode(&t.PageNr)
	if err != nil{
		fmt.Println(err.Error())
		return err
	}
	err = dec.Decode(&t.First)
	if err != nil{
		fmt.Println(err.Error())
		return err
	}
	err = dec.Decode(&t.Last)
	if err != nil{
		fmt.Println(err.Error())
		return err
	}
	fmt.Println("decoded diffRequst", t)
	return nil
}
*/
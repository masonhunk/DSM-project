package multiview

import 	(
	"testing"
	"sync"
	"github.com/stretchr/testify/assert"
	"DSM-project/network"
	"DSM-project/memory"
	"fmt"
	"time"
)

func checkRLockTimeout( lock *sync.RWMutex) bool {
	gotLock := make(chan bool, 1)
	go func() {
		lock.RLock()
		gotLock <- true
		lock.RUnlock()
	}()
	select {
	case  <-gotLock:
		return false
	case <-time.After(time.Second):
		return true
	}
}
func checkWLockTimeout( lock *sync.RWMutex) bool {
	gotLock := make(chan bool, 1)
	go func() {
		lock.Lock()
		gotLock <- true
		lock.Unlock()
	}()
	select {
	case  <-gotLock:
		return false
	case <-time.After(time.Second):
		return true
	}
}

func countChannelCont(c chan bool) int{
	i := 0
	for {
		select {
		case <-c:
			i = i + 1
		default:
			return i
		}
	}
	return i
}

func TestManagerInit(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	NewManager(vmem)
	//TODO just to avoid errors from compiler. Remove at some point.
	fmt.Println("")
}

func TestManager_HandleReadReq(t *testing.T){
	vmem := memory.NewVmem(1024, 128)
	tm := network.NewTranscieverMock()
	m := NewManager( vmem)
	m.tr = tm
	m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:200})
	message, err := m.HandleReadReq(network.Message{Fault_addr: 1100, From: 1, To:3})
	assert.Nil(t, err)
	assert.Equal(t, network.Message{Fault_addr:1100, From:1, To:2, Minipage_base:1024, Minipage_size:128}, message)
}

func TestManager_HandleMultipleReadReq(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	tm := network.NewTranscieverMock()
	m := NewManager(vmem)
	m.tr = tm
	message := network.Message{Fault_addr:1025, From: byte(2), To: byte(1), Type:READ_REQUEST}
	vpage :=  message.Fault_addr / vmem.GetPageSize()
	m.HandleAlloc(network.Message{From: byte(2), To: byte(1), Minipage_size: 1024})
	m.HandleReadReq(message)
	m.HandleReadReq(message)
	assert.True(t, checkWLockTimeout(m.locks[vpage]))
	message.Type = READ_ACK
	m.HandleReadAck(message)
	m.HandleReadAck(message)
	assert.False(t, checkWLockTimeout(m.locks[vpage]))
}

func TestManager_HandleAlloc(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	expmpt := map[int]minipage{}
	tm := network.NewTranscieverMock()
	m := NewManager(vmem)
	m.tr = tm
	m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:200})
	expmpt[8] = minipage{0,128}
	expmpt[9] = minipage{0,72}
	assert.Equal(t, expmpt, m.mpt)
	assert.NotNil(t, m.locks[8])
	assert.Equal(t, 8, m.log[8])
	assert.NotNil(t, m.locks[9])
	assert.Equal(t, 8, m.log[9])
	m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:150})
	expmpt[17] = minipage{72,56}
	expmpt[18] = minipage{0,94}
	assert.Equal(t, expmpt, m.mpt)
	m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:600})
	expmpt[10] = minipage{94,34}
	expmpt[11] = minipage{0,128}
	expmpt[12] = minipage{0,128}
	expmpt[13] = minipage{0,128}
	expmpt[14] = minipage{0,128}
	expmpt[15] = minipage{0,54}
	assert.Equal(t, expmpt, m.mpt)
}

func TestManager_HandleFree(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	expmpt := map[int]minipage{}
	tm := network.NewTranscieverMock()
	m := NewManager(vmem)
	m.tr = tm
	pointer, _ := m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:200})
	expmpt[8] = minipage{0,128}
	expmpt[9] = minipage{0,72}
	assert.Equal(t, expmpt, m.mpt)
	m.HandleFree(network.Message{From:byte(2), To:byte(1), Fault_addr:pointer.Fault_addr})
	assert.Equal(t, 0, len(m.mpt))
	assert.Equal(t, 0, len(m.log))
	assert.Equal(t, 0, len(m.locks))
	assert.Equal(t, 0, len(m.copies))
}

func TestManager_HandleWriteReq(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	tm := network.NewTranscieverMock()
	m := NewManager( vmem)
	m.tr = tm

	message := network.Message{From: byte(2), To: byte(1), Fault_addr: 4}
	reply, err := m.HandleWriteReq(message)
	assert.Error(t, err)
	pointer, _ := m.HandleAlloc(network.Message{From: byte(2), To: byte(1), Minipage_size: 200})
	message.Fault_addr = pointer.Fault_addr+1
	reply, err = m.HandleWriteReq(message)
	assert.Nil(t, err)
	message = network.Message{Fault_addr:message.Fault_addr, From: byte(2), To: byte(2), Minipage_size: 128, Minipage_base: 1024, Privbase:0, Type:INVALIDATE_REQUEST}
	assert.Equal(t, []network.Message{message}, reply)
	message.Type = INVALIDATE_REPLY
	m.HandleInvalidateReply(message)
	vpage :=  message.Fault_addr / vmem.GetPageSize()
	assert.True(t, checkRLockTimeout(m.locks[vpage]))
	message.Type = WRITE_ACK
	m.HandleWriteAck(message)
	assert.False(t, checkRLockTimeout(m.locks[vpage]))
}

func TestManager_HandleMultipleWriteReq(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	tm := network.NewTranscieverMock()
	m := NewManager(vmem)
	m.tr = tm
	pointer, _ := m.HandleAlloc(network.Message{From: byte(2), To: byte(1), Minipage_size: 200})
	message := network.Message{From: byte(2), To: byte(1), Fault_addr: pointer.Fault_addr}

	reply := make(chan bool, 5)

	go func(){
		m.HandleWriteReq(message)
		reply <- true
	}()
	go func(){
		m.HandleWriteReq(message)
		reply <- true
	}()
	time.Sleep(time.Millisecond * 500)
	assert.Equal(t, 1, countChannelCont(reply))
	m.HandleWriteAck(message)
	time.Sleep(time.Millisecond * 500)
	assert.Equal(t, 1, countChannelCont(reply))
	m.HandleWriteAck(message)
	time.Sleep(time.Millisecond * 500)
	vpage :=  message.Fault_addr / vmem.GetPageSize()
	assert.False(t, checkWLockTimeout(m.locks[vpage]))
}
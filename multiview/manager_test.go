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

func checkLockTimeout( lock *sync.RWMutex) bool {
	gotLock := make(chan bool)
	go func() {
		lock.Lock()
		gotLock <- true
		lock.Unlock()
	}()
	select {
	case  <-gotLock:
		return false
	case <-time.After(time.Second * 1):
		return true
	}
}

func TestManagerInit(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	NewManager(network.NewTranscieverMock(),vmem)
	fmt.Println("")
}

func TestManager_HandleReadReq(t *testing.T){
	vmem := memory.NewVmem(1024, 128)
	tm := network.NewTranscieverMock()
	m := NewManager(tm, vmem)
	m.cs.copies[0] = []byte{4}
	m.cs.locks[0] = new(sync.RWMutex)
	m.HandleReadReq(network.Message{Fault_addr: 0, From: 1, To:3})
	assert.Equal(t, []network.Message{{Fault_addr:0, From:1, To:4}}, tm.Messages)
}

func TestManager_HandleAlloc(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	expmpt := map[int]minipage{}
	tm := network.NewTranscieverMock()
	m := NewManager(tm, vmem)
	m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:200})
	expmpt[8] = minipage{0,128}
	expmpt[9] = minipage{0,72}
	assert.Equal(t, expmpt, m.mpt)
	assert.NotNil(t, m.cs.locks[8])
	assert.Equal(t, 8, m.log[8])
	assert.NotNil(t, m.cs.locks[9])
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
	m := NewManager(tm, vmem)
	pointer, _ := m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:200})
	expmpt[8] = minipage{0,128}
	expmpt[9] = minipage{0,72}
	assert.Equal(t, expmpt, m.mpt)
	m.HandleFree(network.Message{From:byte(2), To:byte(1), Fault_addr:pointer.Fault_addr})
	assert.Equal(t, 0, len(m.mpt))
	assert.Equal(t, 0, len(m.log))
	assert.Equal(t, 0, len(m.cs.locks))
	assert.Equal(t, 0, len(m.cs.copies))
}

func TestManager_HandleWriteReq(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	tm := network.NewTranscieverMock()
	m := NewManager(tm, vmem)

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
	assert.True(t, checkLockTimeout(m.cs.locks[vpage]))
	message.Type = WRITE_ACK
	m.HandleWriteAck(message)
	assert.False(t, checkLockTimeout(m.cs.locks[vpage]))
}

/*func TestManager_HandleMultipleWriteReq(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	tm := network.NewTranscieverMock()
	m := NewManager(tm, vmem)
}*/
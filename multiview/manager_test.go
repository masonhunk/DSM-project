package multiview

import 	(
	"testing"
	"sync"
	"github.com/stretchr/testify/assert"
	"DSM-project/network"
	"DSM-project/memory"
	"fmt"
)

func TestManagerInit(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	NewManager(network.NewTranscieverMock(),vmem)
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
	expmpt[128] = minipage{0,128}
	expmpt[129] = minipage{0,72}
	assert.Equal(t, expmpt, m.mpt)
	m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:150})
	expmpt[257] = minipage{72,56}
	expmpt[258] = minipage{0,94}
	assert.Equal(t, expmpt, m.mpt)
	m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:600})
	expmpt[130] = minipage{94,34}
	expmpt[131] = minipage{0,128}
	expmpt[132] = minipage{0,128}
	expmpt[133] = minipage{0,128}
	expmpt[134] = minipage{0,128}
	expmpt[135] = minipage{0,54}
	assert.Equal(t, expmpt, m.mpt)
}

func TestManager_HandleFree(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	expmpt := map[int]minipage{}
	tm := network.NewTranscieverMock()
	m := NewManager(tm, vmem)
	m.HandleAlloc(network.Message{From:byte(2), To:byte(1), Minipage_size:200})
	expmpt[128] = minipage{0,128}
	expmpt[129] = minipage{0,72}
	assert.Equal(t, expmpt, m.mpt)
}
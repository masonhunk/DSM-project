package multiview

import (
	"testing"
	"DSM-project/network"
	"fmt"
	"DSM-project/memory"
	"sync"
	"github.com/stretchr/testify/assert"
)

func TestManagerInit(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	NewManager(network.NewTranscieverMock(),vmem)
}

func TestManager_HandleAlloc(t *testing.T) {

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

func TestMVMem_Malloc(t *testing.T) {
	vmem := memory.NewVmem(1024, 128)
	tm := network.NewTranscieverMock()
	m := NewManager(tm, vmem)
	m.HandleAlloc(200)
	fmt.Println(m.mpt)
	m.HandleAlloc(150)
	fmt.Println(m.mpt)
	m.HandleAlloc(600)
	fmt.Println(m.mpt)

}
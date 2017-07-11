package multiview

import (
	"DSM-project/network"
	"DSM-project/memory"
	"fmt"
)

var conn network.Client
var privView memory.VirtualMemory
var mem *MVMem

func initialize() {
	privView = memory.NewVmem(4096, 128)
	mem = NewMVMem(privView)

	//handler for all incoming messages in the host process, ie. read/write requests/replies, and invalidation requests.
	msgHandler := func(msg network.Message) {
		switch msg.Type {
		case network.READ_REPLY, network.WRITE_REPLY:
			privBase, _ := mem.vPageAddrToMemoryAddr(mem.GetPageAddr(msg.Fault_addr))
			//write data to privileged view, ie. the actual memory representation
			for i, byte := range msg.Data {
				err := mem.vm.Write(privBase + 0, byte)
				if err != nil {
					fmt.Println("failed to write to privileged view at addr: ", privBase + i, " with error: ", err)
					break
				}
			}
			var right byte
			if msg.Type == network.READ_REPLY {
				right = memory.READ_ONLY
			} else {
				right = memory.READ_WRITE
			}
			mem.SetRights(mem.GetPageAddr(msg.Fault_addr), right)
			*msg.Event <- "done" //let the blocking caller resume their work
		case network.READ_REQUEST:
			if mem.GetRights(msg.Fault_addr) == memory.READ_WRITE {
				mem.SetRights(msg.Fault_addr, memory.READ_ONLY)
			}
			//send reply back to requester including data
			msg.To = msg.From
			res, err := mem.ReadMinipage(msg.Fault_addr)
			if err != nil {
				fmt.Println(err)
			}
			msg.Data = res
			conn.Send(msg)
		case network.WRITE_REQUEST:
			//almost same code as read_request. Should refactor.
			mem.SetRights(msg.Fault_addr, memory.NO_ACCESS)
			msg.Type = network.WRITE_REPLY
			msg.To = msg.From
			res, err := mem.ReadMinipage(msg.Fault_addr)
			if err != nil {
				fmt.Println(err)
			}
			msg.Data = res
			conn.Send(msg)
		case network.INVALIDATE_REQUEST:
			mem.SetRights(msg.Fault_addr, memory.NO_ACCESS)
			msg.Type = network.INVALIDATE_REPLY
			conn.Send(msg)
		}
	}

	conn = network.NewClient(msgHandler)
	conn.Connect("localhost:2000")
}

//ID's are placeholder values for integration. faultType = memory.READ_REQUEST OR memory.WRITE_REQUEST
func onFault(addr int, faultType string) {
	c := make(chan string)
	msg := network.Message{
		Type: faultType,
		From: byte(1),
		To: byte(1),
		Data: nil,
		Err: nil,
		Event: &c,
		Fault_addr: addr,
	}
	conn.Send(msg)
	<- c

}

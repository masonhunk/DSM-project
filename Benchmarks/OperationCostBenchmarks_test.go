package Benchmarks

import (
	"testing"
	"DSM-project/multiview"
)

func TestSynchronizedWritesMW(t *testing.T) {
	mw1, mws := setupHosts(1000, 4096,4096)
	addr,_ := mw1.Malloc(1)
	nrRounds := 10000
	for i := 0; i < nrRounds; i++ {
		mw1.Write(addr, byte(1))
		readOnAllHosts(addr, mws)
	}
}

func readOnAllHosts(addr int, mws []*multiview.Multiview) {
	for _,mw := range mws {
		mw.Read(addr)
	}
}

func setupHosts(nrHosts int, memSize, pageByteSize int) (manager *multiview.Multiview, mws []*multiview.Multiview){
	mw1 := multiview.NewMultiView()
	mw1.Initialize(4096, 4096, 2)
	mws = make([]*multiview.Multiview, nrHosts-1)
	for i := range mws {
		mws[i] = multiview.NewMultiView()
		mws[i].Join(4096,4096)
	}
	return
}
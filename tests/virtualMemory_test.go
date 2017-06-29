package tests

import (
	"testing"
	"DSM-project/memory"
	"fmt"
)

func TestNoAccess(t *testing.T) {
	mem := memory.NewVmem(4096, 128)

	r := mem.GetRights(125)
	fmt.Println(r)

	if _, err := mem.Read(125); err == nil {
		t.Error("Expected no access (nil)")
	}
	if err := mem.Write(125, 90); err == nil {
		t.Error("Expected no access (nil)")
	}

}
// Package bpsocket provides BP Socket (AF_BP) address family support for DTN communication
package bpsocket

import (
	"fmt"
	"unsafe"
)

// SockaddrBP represents the kernel's sockaddr_bp structure
type SockaddrBP struct {
	Family  uint16 // AF_BP
	Scheme  int32  // BP_SCHEME_IPN
	NodeNum uint32
	SvcNum  uint32
}

func (sa *SockaddrBP) ToBytes() []byte {
	size := int(unsafe.Sizeof(*sa))
	ptr := unsafe.Pointer(sa)
	return (*[unsafe.Sizeof(SockaddrBP{})]byte)(ptr)[:size:size]
}

func NewSockaddrBP(nodeNum, svcNum uint64) *SockaddrBP {
	return &SockaddrBP{
		Family:  AF_BP,
		Scheme:  BP_SCHEME_IPN,
		NodeNum: uint32(nodeNum),
		SvcNum:  uint32(svcNum),
	}
}

func (sa *SockaddrBP) String() string {
	return fmt.Sprintf("ipn:%d.%d", sa.NodeNum, sa.SvcNum)
}

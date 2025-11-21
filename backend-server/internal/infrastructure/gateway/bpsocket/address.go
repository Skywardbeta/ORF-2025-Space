// address.go - IPNアドレス（ipn:node.service）の構造体と変換処理
package bpsocket

import (
	"fmt"
	"unsafe"
)

// カーネルモジュールのsockaddr_bp構造体に対応（uint16 family, uint64 node, uint64 service）
type SockaddrBP struct {
	Family  uint16
	NodeNum uint64
	SvcNum  uint64
}

func (sa *SockaddrBP) ToBytes() []byte {
	size := int(unsafe.Sizeof(*sa))
	ptr := unsafe.Pointer(sa)
	return (*[unsafe.Sizeof(SockaddrBP{})]byte)(ptr)[:size:size]
}

func NewSockaddrBP(nodeNum, svcNum uint64) *SockaddrBP {
	return &SockaddrBP{
		Family:  AF_BP,
		NodeNum: nodeNum,
		SvcNum:  svcNum,
	}
}

func (sa *SockaddrBP) String() string {
	return fmt.Sprintf("ipn:%d.%d", sa.NodeNum, sa.SvcNum)
}

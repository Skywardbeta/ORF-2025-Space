//go:build windows
// +build windows

// Package bpsocket provides Windows stubs (BP Socket not supported on Windows)
package bpsocket

import (
	"fmt"
	"syscall"
)

func closeFd(fd int) error {
	return syscall.Close(syscall.Handle(fd))
}

func bind(fd int, addr *SockaddrBP) error {
	return fmt.Errorf("bp-socket not supported on Windows")
}

func sendto(fd int, data []byte, remoteAddr *SockaddrBP) error {
	return fmt.Errorf("bp-socket not supported on Windows")
}

func recvfrom(fd int, buf []byte) (int, *SockaddrBP, error) {
	return 0, nil, fmt.Errorf("bp-socket not supported on Windows")
}

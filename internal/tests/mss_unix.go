//go:build !windows

package tests

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"
)

func observeMSSWithSocket(ctx context.Context, target, network string) (int, error) {
	address, err := normalizeAddress(target, "443")
	if err != nil {
		return 0, err
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return 0, errors.New("non-TCP connection")
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var mss int
	var sysErr error
	controlErr := rawConn.Control(func(fd uintptr) {
		mss, sysErr = syscall.GetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_MAXSEG)
	})
	if controlErr != nil {
		return 0, controlErr
	}
	if sysErr != nil {
		return 0, sysErr
	}
	if mss <= 0 {
		return 0, errors.New("invalid MSS result")
	}
	return mss, nil
}

package netpolicy

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// Assert the real Linux socket protocol, not just Go's configuration getter.
func TestLinuxListenersUsePlainTCP(t *testing.T) {
	t.Setenv("GODEBUG", "multipathtcp=1")
	var before net.ListenConfig
	if !before.MultipathTCP() {
		t.Fatal("test must start with MPTCP enabled in Go")
	}
	if err := Enforce(); err != nil {
		t.Fatal(err)
	}
	for _, network := range []string{"tcp4", "tcp6"} {
		t.Run(network, func(t *testing.T) {
			var config net.ListenConfig
			listener, err := config.Listen(context.Background(), network, ":0")
			if errors.Is(err, syscall.EAFNOSUPPORT) || errors.Is(err, syscall.EPROTONOSUPPORT) {
				t.Skipf("%s unavailable: %v", network, err)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			raw, err := listener.(*net.TCPListener).SyscallConn()
			if err != nil {
				t.Fatal(err)
			}
			var protocol int
			var socketErr error
			if err := raw.Control(func(fd uintptr) {
				protocol, socketErr = unix.GetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_PROTOCOL)
			}); err != nil {
				t.Fatal(err)
			}
			if socketErr != nil || protocol != unix.IPPROTO_TCP {
				t.Fatalf("listener protocol = %d, err = %v, want TCP (%d)", protocol, socketErr, unix.IPPROTO_TCP)
			}
		})
	}
}

package errors

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"syscall"
)

// IsRetryable — مشخص میکنه آیا error قابل retry هست یا نه
func IsRetryable(err error) bool {
	if err == nil || err == io.EOF {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		errno := syscallErr.Err
		return errno == syscall.ECONNRESET ||
			errno == syscall.EPIPE ||
			errno == syscall.ECONNABORTED ||
			errno == syscall.ETIMEDOUT ||
			errno == syscall.ECONNREFUSED ||
			errno == syscall.EHOSTUNREACH ||
			errno == syscall.ENETUNREACH
	}

	return true
}

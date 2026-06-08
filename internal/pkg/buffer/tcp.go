package buffer

import (
	"io"
)

func CopyT(dst io.Writer, src io.Reader) error {
	buf := make([]byte, tcpBufSize())

	_, err := io.CopyBuffer(dst, src, buf)
	return err
}

package tcp

import "io"

func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)

	// buf := make([]byte, 256)
	// return io.CopyBuffer(dst, src, buf)
}

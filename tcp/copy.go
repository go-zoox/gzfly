package tcp

import "io"

func Copy(dst io.WriteCloser, src io.ReadCloser) (written int64, err error) {
	defer src.Close()
	defer dst.Close()

	return io.Copy(dst, src)

	// buf := make([]byte, 256)
	// return io.CopyBuffer(dst, src, buf)
}

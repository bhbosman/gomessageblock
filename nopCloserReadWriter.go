package gomessageblock

import "io"

type nopCloserReadWriter struct {
	io.ReadWriter
}

func (nopCloserReadWriter) Close() error { return nil }

// NopCloser returns a ReadCloser with a no-op Close method wrapping
// the provided ioReader r.
func NopCloserReadWriter(rw io.ReadWriter) nopCloserReadWriter {
	return nopCloserReadWriter{rw}
}

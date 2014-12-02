package io

import "io"

// LimitedReadWriter is similar to io.LimitedReader; it writes to RW but
// limits the amount of data written to just N bytes. Each subsequent call to
// Write() will return a nil error and a count of 0.
type LimitedReadWriter struct {
	RW io.ReadWriter
	N  int64
}

func (l *LimitedReadWriter) Write(data []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, nil
	}
	if int64(len(data)) > l.N {
		data = data[0:l.N]
	}
	n, err = l.RW.Write(data)
	l.N -= int64(n)
	return
}

func (l *LimitedReadWriter) Read(p []byte) (n int, err error) {
	return l.RW.Read(p)
}

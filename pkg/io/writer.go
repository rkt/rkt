package io

import "io"

// LimitedWriter is similar to io.LimitedReader; it writes to W but
// limits the amount of data written to just N bytes. Each subsequent call to
// Write() will return a nil error and a count of 0.
type LimitedWriter struct {
	W io.Writer
	N int64
}

func (l *LimitedWriter) Write(data []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, nil
	}
	if int64(len(data)) > l.N {
		data = data[0:l.N]
	}
	n, err = l.W.Write(data)
	l.N -= int64(n)
	return
}

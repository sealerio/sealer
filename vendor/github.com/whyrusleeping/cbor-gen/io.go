package typegen

import (
	"io"
)

var (
	_ io.Reader      = (*CborReader)(nil)
	_ io.ByteScanner = (*CborReader)(nil)
)

type CborReader struct {
	r    BytePeeker
	hbuf []byte
}

func NewCborReader(r io.Reader) *CborReader {
	if r, ok := r.(*CborReader); ok {
		return r
	}

	return &CborReader{
		r:    GetPeeker(r),
		hbuf: make([]byte, maxHeaderSize),
	}
}

func (cr *CborReader) Read(p []byte) (n int, err error) {
	return cr.r.Read(p)
}

func (cr *CborReader) ReadByte() (byte, error) {
	return cr.r.ReadByte()
}

func (cr *CborReader) UnreadByte() error {
	return cr.r.UnreadByte()
}

func (cr *CborReader) ReadHeader() (byte, uint64, error) {
	return CborReadHeaderBuf(cr.r, cr.hbuf)
}

func (cr *CborReader) SetReader(r io.Reader) {
	cr.r = GetPeeker(r)
}

var (
	_ io.Writer       = (*CborWriter)(nil)
	_ io.StringWriter = (*CborWriter)(nil)
)

type CborWriter struct {
	w    io.Writer
	hbuf []byte
}

func NewCborWriter(w io.Writer) *CborWriter {
	if w, ok := w.(*CborWriter); ok {
		return w
	}
	return &CborWriter{
		w:    w,
		hbuf: make([]byte, maxHeaderSize),
	}
}

func (cw *CborWriter) Write(p []byte) (n int, err error) {
	return cw.w.Write(p)
}

func (cw *CborWriter) WriteMajorTypeHeader(t byte, l uint64) error {
	return WriteMajorTypeHeaderBuf(cw.hbuf, cw.w, t, l)
}

func (cw *CborWriter) CborWriteHeader(t byte, l uint64) error {
	return WriteMajorTypeHeaderBuf(cw.hbuf, cw.w, t, l)
}

func (cw *CborWriter) WriteString(s string) (int, error) {
	if sw, ok := cw.w.(io.StringWriter); ok {
		return sw.WriteString(s)
	}
	return cw.w.Write([]byte(s))
}

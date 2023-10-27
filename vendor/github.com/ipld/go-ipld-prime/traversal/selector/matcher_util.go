package selector

import (
	"io"
)

type readerat struct {
	rs  io.ReadSeeker
	off int64
}

// ReadAt provides the io.ReadAt method over a ReadSeeker. It will track the
// current offset and seek if necessary.
func (r *readerat) ReadAt(p []byte, off int64) (n int, err error) {
	if off != r.off {
		if _, err = r.rs.Seek(off, io.SeekStart); err != nil {
			return 0, err
		}
		r.off = off
	}
	c, err := r.rs.Read(p)
	if err != nil {
		return c, err
	}
	r.off += int64(c)
	return c, nil
}

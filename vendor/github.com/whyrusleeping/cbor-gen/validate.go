package typegen

import (
	"bytes"
	"fmt"
	"io"
)

// ValidateCBOR validates that a byte array is a single valid CBOR object.
func ValidateCBOR(b []byte) error {
	// The code here is basically identical to the previous function, it
	// just doesn't copy.

	br := bytes.NewReader(b)

	for remaining := uint64(1); remaining > 0; remaining-- {
		maj, extra, err := CborReadHeader(br)
		if err != nil {
			return err
		}

		switch maj {
		case MajUnsignedInt, MajNegativeInt, MajOther:
			// nothing fancy to do
		case MajByteString, MajTextString:
			if extra > ByteArrayMaxLen {
				return maxLengthError
			}
			if uint64(br.Len()) < extra {
				return io.ErrUnexpectedEOF
			}

			if _, err := br.Seek(int64(extra), io.SeekCurrent); err != nil {
				return err
			}
		case MajTag:
			remaining++
		case MajArray:
			if extra > MaxLength {
				return maxLengthError
			}
			remaining += extra
		case MajMap:
			if extra > MaxLength {
				return maxLengthError
			}
			remaining += extra * 2
		default:
			return fmt.Errorf("unhandled deferred cbor type: %d", maj)
		}
	}
	if br.Len() > 0 {
		return fmt.Errorf("unexpected %d unread bytes", br.Len())
	}
	return nil
}

package metadata

import (
	"fmt"
	"io"

	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf

func (t *Item) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{162}); err != nil {
		return err
	}

	scratch := make([]byte, 9)

	// t.Link (cid.Cid) (struct)
	if len("link") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"link\" was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajTextString, uint64(len("link"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "link"); err != nil {
		return err
	}

	if err := cbg.WriteCidBuf(scratch, w, t.Link); err != nil {
		return xerrors.Errorf("failed to write cid field t.Link: %w", err)
	}

	// t.BlockPresent (bool) (bool)
	if len("blockPresent") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"blockPresent\" was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajTextString, uint64(len("blockPresent"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "blockPresent"); err != nil {
		return err
	}

	if err := cbg.WriteBool(w, t.BlockPresent); err != nil {
		return err
	}
	return nil
}

func (t *Item) UnmarshalCBOR(r io.Reader) error {
	*t = Item{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("Item: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadStringBuf(br, scratch)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.Link (cid.Cid) (struct)
		case "link":

			{

				c, err := cbg.ReadCid(br)
				if err != nil {
					return xerrors.Errorf("failed to read cid field t.Link: %w", err)
				}

				t.Link = c

			}
			// t.BlockPresent (bool) (bool)
		case "blockPresent":

			maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
			if err != nil {
				return err
			}
			if maj != cbg.MajOther {
				return fmt.Errorf("booleans must be major type 7")
			}
			switch extra {
			case 20:
				t.BlockPresent = false
			case 21:
				t.BlockPresent = true
			default:
				return fmt.Errorf("booleans are either major type 7, value 20 or 21 (got %d)", extra)
			}

		default:
			return fmt.Errorf("unknown struct field %d: '%s'", i, name)
		}
	}

	return nil
}

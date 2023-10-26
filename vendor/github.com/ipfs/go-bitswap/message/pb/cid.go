package bitswap_message_pb

import (
	"github.com/ipfs/go-cid"
)

// NOTE: Don't "embed" the cid, wrap it like we're doing here. Otherwise, gogo
// will try to use the Bytes() function.

// Cid is a custom type for CIDs in protobufs, that allows us to avoid
// reallocating.
type Cid struct {
	Cid cid.Cid
}

func (c Cid) Marshal() ([]byte, error) {
	return c.Cid.Bytes(), nil
}

func (c *Cid) MarshalTo(data []byte) (int, error) {
	// intentionally using KeyString here to avoid allocating.
	return copy(data[:c.Size()], c.Cid.KeyString()), nil
}

func (c *Cid) Unmarshal(data []byte) (err error) {
	c.Cid, err = cid.Cast(data)
	return err
}

func (c *Cid) Size() int {
	return len(c.Cid.KeyString())
}

func (c Cid) MarshalJSON() ([]byte, error) {
	return c.Cid.MarshalJSON()
}

func (c *Cid) UnmarshalJSON(data []byte) error {
	return c.Cid.UnmarshalJSON(data)
}

func (c Cid) Equal(other Cid) bool {
	return c.Cid.Equals(c.Cid)
}

package ipns

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/ipld/go-ipld-prime"
	_ "github.com/ipld/go-ipld-prime/codec/dagcbor" // used to import the DagCbor encoder/decoder
	ipldcodec "github.com/ipld/go-ipld-prime/multicodec"
	"github.com/ipld/go-ipld-prime/node/basic"

	"github.com/multiformats/go-multicodec"

	"github.com/gogo/protobuf/proto"

	pb "github.com/ipfs/go-ipns/pb"

	u "github.com/ipfs/go-ipfs-util"
	ic "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

const (
	validity     = "Validity"
	validityType = "ValidityType"
	value        = "Value"
	sequence     = "Sequence"
	ttl          = "TTL"
)

// Create creates a new IPNS entry and signs it with the given private key.
//
// This function does not embed the public key. If you want to do that, use
// `EmbedPublicKey`.
func Create(sk ic.PrivKey, val []byte, seq uint64, eol time.Time, ttl time.Duration) (*pb.IpnsEntry, error) {
	entry := new(pb.IpnsEntry)

	entry.Value = val
	typ := pb.IpnsEntry_EOL
	entry.ValidityType = &typ
	entry.Sequence = &seq
	entry.Validity = []byte(u.FormatRFC3339(eol))

	ttlNs := uint64(ttl.Nanoseconds())
	entry.Ttl = proto.Uint64(ttlNs)

	cborData, err := createCborDataForIpnsEntry(entry)
	if err != nil {
		return nil, err
	}
	entry.Data = cborData

	sig1, err := sk.Sign(ipnsEntryDataForSigV1(entry))
	if err != nil {
		return nil, errors.Wrap(err, "could not compute signature data")
	}
	entry.SignatureV1 = sig1

	sig2Data, err := ipnsEntryDataForSigV2(entry)
	if err != nil {
		return nil, err
	}
	sig2, err := sk.Sign(sig2Data)
	if err != nil {
		return nil, err
	}
	entry.SignatureV2 = sig2

	return entry, nil
}

func createCborDataForIpnsEntry(e *pb.IpnsEntry) ([]byte, error) {
	m := make(map[string]ipld.Node)
	var keys []string
	m[value] = basicnode.NewBytes(e.GetValue())
	keys = append(keys, value)

	m[validity] = basicnode.NewBytes(e.GetValidity())
	keys = append(keys, validity)

	m[validityType] = basicnode.NewInt(int64(e.GetValidityType()))
	keys = append(keys, validityType)

	m[sequence] = basicnode.NewInt(int64(e.GetSequence()))
	keys = append(keys, sequence)

	m[ttl] = basicnode.NewInt(int64(e.GetTtl()))
	keys = append(keys, ttl)

	sort.Sort(cborMapKeyString_RFC7049(keys))

	newNd := basicnode.Prototype__Map{}.NewBuilder()
	ma, err := newNd.BeginMap(int64(len(keys)))
	if err != nil {
		return nil, err
	}

	for _, k := range keys {
		if err := ma.AssembleKey().AssignString(k); err != nil {
			return nil, err
		}
		if err := ma.AssembleValue().AssignNode(m[k]); err != nil {
			return nil, err
		}
	}

	if err := ma.Finish(); err != nil {
		return nil, err
	}

	nd := newNd.Build()

	enc, err := ipldcodec.LookupEncoder(uint64(multicodec.DagCbor))
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := enc(nd, buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Validates validates the given IPNS entry against the given public key.
func Validate(pk ic.PubKey, entry *pb.IpnsEntry) error {
	// Check the ipns record signature with the public key

	// Check v2 signature if it's available, otherwise use the v1 signature
	if entry.GetSignatureV2() != nil {
		sig2Data, err := ipnsEntryDataForSigV2(entry)
		if err != nil {
			return fmt.Errorf("could not compute signature data: %w", err)
		}
		if ok, err := pk.Verify(sig2Data, entry.GetSignatureV2()); err != nil || !ok {
			return ErrSignature
		}

		// TODO: If we switch from pb.IpnsEntry to a more generic IpnsRecord type then perhaps we should only check
		// this if there is no v1 signature. In the meanwhile this helps avoid some potential rough edges around people
		// checking the entry fields instead of doing CBOR decoding everywhere.
		if err := validateCborDataMatchesPbData(entry); err != nil {
			return err
		}
	} else {
		if ok, err := pk.Verify(ipnsEntryDataForSigV1(entry), entry.GetSignatureV1()); err != nil || !ok {
			return ErrSignature
		}
	}

	eol, err := GetEOL(entry)
	if err != nil {
		return err
	}
	if time.Now().After(eol) {
		return ErrExpiredRecord
	}
	return nil
}

// TODO: Most of this function could probably be replaced with codegen
func validateCborDataMatchesPbData(entry *pb.IpnsEntry) error {
	if len(entry.GetData()) == 0 {
		return fmt.Errorf("record data is missing")
	}

	dec, err := ipldcodec.LookupDecoder(uint64(multicodec.DagCbor))
	if err != nil {
		return err
	}

	ndbuilder := basicnode.Prototype__Map{}.NewBuilder()
	if err := dec(ndbuilder, bytes.NewReader(entry.GetData())); err != nil {
		return err
	}

	fullNd := ndbuilder.Build()
	nd, err := fullNd.LookupByString(value)
	if err != nil {
		return err
	}
	ndBytes, err := nd.AsBytes()
	if err != nil {
		return err
	}
	if !bytes.Equal(entry.GetValue(), ndBytes) {
		return fmt.Errorf("field \"%v\" did not match between protobuf and CBOR", value)
	}

	nd, err = fullNd.LookupByString(validity)
	if err != nil {
		return err
	}
	ndBytes, err = nd.AsBytes()
	if err != nil {
		return err
	}
	if !bytes.Equal(entry.GetValidity(), ndBytes) {
		return fmt.Errorf("field \"%v\" did not match between protobuf and CBOR", validity)
	}

	nd, err = fullNd.LookupByString(validityType)
	if err != nil {
		return err
	}
	ndInt, err := nd.AsInt()
	if err != nil {
		return err
	}
	if int64(entry.GetValidityType()) != ndInt {
		return fmt.Errorf("field \"%v\" did not match between protobuf and CBOR", validityType)
	}

	nd, err = fullNd.LookupByString(sequence)
	if err != nil {
		return err
	}
	ndInt, err = nd.AsInt()
	if err != nil {
		return err
	}

	if entry.GetSequence() != uint64(ndInt) {
		return fmt.Errorf("field \"%v\" did not match between protobuf and CBOR", sequence)
	}

	nd, err = fullNd.LookupByString("TTL")
	if err != nil {
		return err
	}
	ndInt, err = nd.AsInt()
	if err != nil {
		return err
	}
	if entry.GetTtl() != uint64(ndInt) {
		return fmt.Errorf("field \"%v\" did not match between protobuf and CBOR", ttl)
	}

	return nil
}

// GetEOL returns the EOL of this IPNS entry
//
// This function returns ErrUnrecognizedValidity if the validity type of the
// record isn't EOL. Otherwise, it returns an error if it can't parse the EOL.
func GetEOL(entry *pb.IpnsEntry) (time.Time, error) {
	if entry.GetValidityType() != pb.IpnsEntry_EOL {
		return time.Time{}, ErrUnrecognizedValidity
	}
	return u.ParseRFC3339(string(entry.GetValidity()))
}

// EmbedPublicKey embeds the given public key in the given ipns entry. While not
// strictly required, some nodes (e.g., DHT servers) may reject IPNS entries
// that don't embed their public keys as they may not be able to validate them
// efficiently.
func EmbedPublicKey(pk ic.PubKey, entry *pb.IpnsEntry) error {
	// Try extracting the public key from the ID. If we can, *don't* embed
	// it.
	id, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return err
	}
	if _, err := id.ExtractPublicKey(); err != peer.ErrNoPublicKey {
		// Either a *real* error or nil.
		return err
	}

	// We failed to extract the public key from the peer ID, embed it in the
	// record.
	pkBytes, err := ic.MarshalPublicKey(pk)
	if err != nil {
		return err
	}
	entry.PubKey = pkBytes
	return nil
}

// ExtractPublicKey extracts a public key matching `pid` from the IPNS record,
// if possible.
//
// This function returns (nil, nil) when no public key can be extracted and
// nothing is malformed.
func ExtractPublicKey(pid peer.ID, entry *pb.IpnsEntry) (ic.PubKey, error) {
	if entry.PubKey != nil {
		pk, err := ic.UnmarshalPublicKey(entry.PubKey)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling pubkey in record: %s", err)
		}

		expPid, err := peer.IDFromPublicKey(pk)
		if err != nil {
			return nil, fmt.Errorf("could not regenerate peerID from pubkey: %s", err)
		}

		if pid != expPid {
			return nil, ErrPublicKeyMismatch
		}
		return pk, nil
	}

	return pid.ExtractPublicKey()
}

// Compare compares two IPNS entries. It returns:
//
// * -1 if a is older than b
// * 0 if a and b cannot be ordered (this doesn't mean that they are equal)
// * +1 if a is newer than b
//
// It returns an error when either a or b are malformed.
//
// NOTE: It *does not* validate the records, the caller is responsible for calling
// `Validate` first.
//
// NOTE: If a and b cannot be ordered by this function, you can determine their
// order by comparing their serialized byte representations (using
// `bytes.Compare`). You must do this if you are implementing a libp2p record
// validator (or you can just use the one provided for you by this package).
func Compare(a, b *pb.IpnsEntry) (int, error) {
	aHasV2Sig := a.GetSignatureV2() != nil
	bHasV2Sig := b.GetSignatureV2() != nil

	// Having a newer signature version is better than an older signature version
	if aHasV2Sig && !bHasV2Sig {
		return 1, nil
	} else if !aHasV2Sig && bHasV2Sig {
		return -1, nil
	}

	as := a.GetSequence()
	bs := b.GetSequence()

	if as > bs {
		return 1, nil
	} else if as < bs {
		return -1, nil
	}

	at, err := u.ParseRFC3339(string(a.GetValidity()))
	if err != nil {
		return 0, err
	}

	bt, err := u.ParseRFC3339(string(b.GetValidity()))
	if err != nil {
		return 0, err
	}

	if at.After(bt) {
		return 1, nil
	} else if bt.After(at) {
		return -1, nil
	}

	return 0, nil
}

func ipnsEntryDataForSigV1(e *pb.IpnsEntry) []byte {
	return bytes.Join([][]byte{
		e.Value,
		e.Validity,
		[]byte(fmt.Sprint(e.GetValidityType())),
	},
		[]byte{})
}

func ipnsEntryDataForSigV2(e *pb.IpnsEntry) ([]byte, error) {
	dataForSig := []byte("ipns-signature:")
	dataForSig = append(dataForSig, e.Data...)

	return dataForSig, nil
}

type cborMapKeyString_RFC7049 []string

func (x cborMapKeyString_RFC7049) Len() int      { return len(x) }
func (x cborMapKeyString_RFC7049) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x cborMapKeyString_RFC7049) Less(i, j int) bool {
	li, lj := len(x[i]), len(x[j])
	if li == lj {
		return x[i] < x[j]
	}
	return li < lj
}

var _ sort.Interface = (cborMapKeyString_RFC7049)(nil)

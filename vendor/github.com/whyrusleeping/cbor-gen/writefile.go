package typegen

import (
	"bytes"
	"go/format"
	"os"

	"golang.org/x/xerrors"
)

// WriteTupleFileEncodersToFile generates array backed MarshalCBOR and UnmarshalCBOR implementations for the
// given types in the specified file, with the specified package name.
//
// The MarshalCBOR and UnmarshalCBOR implementations will marshal/unmarshal each type's fields as a
// fixed-length CBOR array of field values.
func WriteTupleEncodersToFile(fname, pkg string, types ...interface{}) error {
	buf := new(bytes.Buffer)

	typeInfos := make([]*GenTypeInfo, len(types))
	for i, t := range types {
		gti, err := ParseTypeInfo(t)
		if err != nil {
			return xerrors.Errorf("failed to parse type info: %w", err)
		}
		typeInfos[i] = gti
	}

	if err := PrintHeaderAndUtilityMethods(buf, pkg, typeInfos); err != nil {
		return xerrors.Errorf("failed to write header: %w", err)
	}

	for _, t := range typeInfos {
		if err := GenTupleEncodersForType(t, buf); err != nil {
			return xerrors.Errorf("failed to generate encoders: %w", err)
		}
	}

	data, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	fi, err := os.Create(fname)
	if err != nil {
		return xerrors.Errorf("failed to open file: %w", err)
	}

	_, err = fi.Write(data)
	if err != nil {
		_ = fi.Close()
		return err
	}
	_ = fi.Close()

	return nil
}

// WriteMapFileEncodersToFile generates map backed MarshalCBOR and UnmarshalCBOR implementations for
// the given types in the specified file, with the specified package name.
//
// The MarshalCBOR and UnmarshalCBOR implementations will marshal/unmarshal each type's fields as a
// map of field names to field values.
func WriteMapEncodersToFile(fname, pkg string, types ...interface{}) error {
	buf := new(bytes.Buffer)

	typeInfos := make([]*GenTypeInfo, len(types))
	for i, t := range types {
		gti, err := ParseTypeInfo(t)
		if err != nil {
			return xerrors.Errorf("failed to parse type info: %w", err)
		}
		typeInfos[i] = gti
	}

	if err := PrintHeaderAndUtilityMethods(buf, pkg, typeInfos); err != nil {
		return xerrors.Errorf("failed to write header: %w", err)
	}

	for _, t := range typeInfos {
		if err := GenMapEncodersForType(t, buf); err != nil {
			return xerrors.Errorf("failed to generate encoders: %w", err)
		}
	}

	data, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	fi, err := os.Create(fname)
	if err != nil {
		return xerrors.Errorf("failed to open file: %w", err)
	}

	_, err = fi.Write(data)
	if err != nil {
		_ = fi.Close()
		return err
	}
	_ = fi.Close()

	return nil
}

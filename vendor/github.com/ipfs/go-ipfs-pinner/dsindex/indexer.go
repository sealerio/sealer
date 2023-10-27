// Package dsindex provides secondary indexing functionality for a datastore.
package dsindex

import (
	"context"
	"fmt"
	"path"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"github.com/ipfs/go-datastore/query"
	"github.com/multiformats/go-multibase"
)

// Indexer maintains a secondary index.  An index is a collection of key-value
// mappings where the key is the secondary index that maps to one or more
// values, where each value is a unique key being indexed.
type Indexer interface {
	// Add adds the specified value to the key
	Add(ctx context.Context, key, value string) error

	// Delete deletes the specified value from the key.  If the value is not in
	// the datastore, this method returns no error.
	Delete(ctx context.Context, key, value string) error

	// DeleteKey deletes all values in the given key.  If a key is not in the
	// datastore, this method returns no error.  Returns a count of values that
	// were deleted.
	DeleteKey(ctx context.Context, key string) (count int, err error)

	// DeleteAll deletes all keys managed by this Indexer.  Returns a count of
	// the values that were deleted.
	DeleteAll(ctx context.Context) (count int, err error)

	// ForEach calls the function for each value in the specified key, until
	// there are no more values, or until the function returns false.  If key
	// is empty string, then all keys are iterated.
	ForEach(ctx context.Context, key string, fn func(key, value string) bool) error

	// HasValue determines if the key contains the specified value
	HasValue(ctx context.Context, key, value string) (bool, error)

	// HasAny determines if any value is in the specified key.  If key is
	// empty string, then all values are searched.
	HasAny(ctx context.Context, key string) (bool, error)

	// Search returns all values for the given key
	Search(ctx context.Context, key string) (values []string, err error)
}

// indexer is a simple implementation of Indexer.  This implementation relies
// on the underlying data store to support efficient querying by prefix.
//
// TODO: Consider adding caching
type indexer struct {
	dstore ds.Datastore
}

// New creates a new datastore index.  All indexes are stored under the
// specified index name.
//
// To persist the actions of calling Indexer functions, it is necessary to call
// dstore.Sync.
func New(dstore ds.Datastore, name ds.Key) Indexer {
	return &indexer{
		dstore: namespace.Wrap(dstore, name),
	}
}

func (x *indexer) Add(ctx context.Context, key, value string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if value == "" {
		return ErrEmptyValue
	}
	dsKey := ds.NewKey(encode(key)).ChildString(encode(value))
	return x.dstore.Put(ctx, dsKey, []byte{})
}

func (x *indexer) Delete(ctx context.Context, key, value string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if value == "" {
		return ErrEmptyValue
	}
	return x.dstore.Delete(ctx, ds.NewKey(encode(key)).ChildString(encode(value)))
}

func (x *indexer) DeleteKey(ctx context.Context, key string) (int, error) {
	if key == "" {
		return 0, ErrEmptyKey
	}
	return x.deletePrefix(ctx, encode(key))
}

func (x *indexer) DeleteAll(ctx context.Context) (int, error) {
	return x.deletePrefix(ctx, "")
}

func (x *indexer) ForEach(ctx context.Context, key string, fn func(key, value string) bool) error {
	if key != "" {
		key = encode(key)
	}

	q := query.Query{
		Prefix:   key,
		KeysOnly: true,
	}
	results, err := x.dstore.Query(ctx, q)
	if err != nil {
		return err
	}
	defer results.Close()

	for r := range results.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if r.Error != nil {
			return fmt.Errorf("cannot read index: %v", r.Error)
		}
		ent := r.Entry
		decIdx, err := decode(path.Base(path.Dir(ent.Key)))
		if err != nil {
			return fmt.Errorf("cannot decode index: %v", err)
		}
		decKey, err := decode(path.Base(ent.Key))
		if err != nil {
			return fmt.Errorf("cannot decode key: %v", err)
		}
		if !fn(decIdx, decKey) {
			return nil
		}
	}

	return nil
}

func (x *indexer) HasValue(ctx context.Context, key, value string) (bool, error) {
	if key == "" {
		return false, ErrEmptyKey
	}
	if value == "" {
		return false, ErrEmptyValue
	}
	return x.dstore.Has(ctx, ds.NewKey(encode(key)).ChildString(encode(value)))
}

func (x *indexer) HasAny(ctx context.Context, key string) (bool, error) {
	var any bool
	err := x.ForEach(ctx, key, func(key, value string) bool {
		any = true
		return false
	})
	return any, err
}

func (x *indexer) Search(ctx context.Context, key string) ([]string, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}
	ents, err := x.queryPrefix(ctx, encode(key))
	if err != nil {
		return nil, err
	}
	if len(ents) == 0 {
		return nil, nil
	}

	values := make([]string, len(ents))
	for i := range ents {
		values[i], err = decode(path.Base(ents[i].Key))
		if err != nil {
			return nil, fmt.Errorf("cannot decode value: %v", err)
		}
	}
	return values, nil
}

// SyncIndex synchronizes the keys in the target Indexer to match those of the
// ref Indexer. This function does not change this indexer's key root (name
// passed into New).
func SyncIndex(ctx context.Context, ref, target Indexer) (bool, error) {
	// Build reference index map
	refs := map[string]string{}
	err := ref.ForEach(ctx, "", func(key, value string) bool {
		refs[value] = key
		return true
	})
	if err != nil {
		return false, err
	}
	if len(refs) == 0 {
		return false, nil
	}

	// Compare current indexes
	dels := map[string]string{}
	err = target.ForEach(ctx, "", func(key, value string) bool {
		refKey, ok := refs[value]
		if ok && refKey == key {
			// same in both; delete from refs, do not add to dels
			delete(refs, value)
		} else {
			dels[value] = key
		}
		return true
	})
	if err != nil {
		return false, err
	}

	// Items in dels are keys that no longer exist
	for value, key := range dels {
		err = target.Delete(ctx, key, value)
		if err != nil {
			return false, err
		}
	}

	// What remains in refs are keys that need to be added
	for value, key := range refs {
		err = target.Add(ctx, key, value)
		if err != nil {
			return false, err
		}
	}

	return len(refs) != 0 || len(dels) != 0, nil
}

func (x *indexer) deletePrefix(ctx context.Context, prefix string) (int, error) {
	ents, err := x.queryPrefix(ctx, prefix)
	if err != nil {
		return 0, err
	}

	for i := range ents {
		err = x.dstore.Delete(ctx, ds.NewKey(ents[i].Key))
		if err != nil {
			return 0, err
		}
	}

	return len(ents), nil
}

func (x *indexer) queryPrefix(ctx context.Context, prefix string) ([]query.Entry, error) {
	q := query.Query{
		Prefix:   prefix,
		KeysOnly: true,
	}
	results, err := x.dstore.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	return results.Rest()
}

func encode(data string) string {
	encData, err := multibase.Encode(multibase.Base64url, []byte(data))
	if err != nil {
		// programming error; using unsupported encoding
		panic(err.Error())
	}
	return encData
}

func decode(data string) (string, error) {
	_, b, err := multibase.Decode(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

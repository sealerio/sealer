package version

import "fmt"

// Collection is a type that implements the sort.Interface interface
// so that versions can be sorted.
type Collection []*Version

func NewCollection(versions ...string) (Collection, error) {
	c := make(Collection, len(versions))
	for i, v := range versions {
		nv, err := NewVersion(v)
		if err != nil {
			return Collection{}, fmt.Errorf("invalid version '%s': %w", v, err)
		}
		c[i] = nv
	}
	return c, nil
}

func (v Collection) Len() int {
	return len(v)
}

func (v Collection) Less(i, j int) bool {
	return v[i].Compare(v[j]) < 0
}

func (v Collection) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

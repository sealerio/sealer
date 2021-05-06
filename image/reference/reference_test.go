package reference

import (
	"fmt"
	"testing"
)

func TestParseToNamed(t *testing.T) {
	type namedTest struct {
		name    string
		desired Named
	}

	ts := []namedTest{
		{
			name: "xxx.com/abc/tag:v1",
			desired: Named{
				raw:     "xxx.com/abc/tag:v1",
				domain:  "xxx.com",
				repo:    "abc/tag",
				tag:     "v1",
				repoTag: "abc/tag:v1",
			},
		},
		{
			name: "abc/tag:v1",
			desired: Named{
				raw:     "abc/tag:v1",
				domain:  defaultDomain,
				repo:    "abc/tag",
				tag:     "v1",
				repoTag: "abc/tag:v1",
			},
		},
		{
			name: "tag:v1",
			desired: Named{
				raw:     "tag:v1",
				domain:  defaultDomain,
				repo:    defaultRepo + "/tag",
				tag:     "v1",
				repoTag: defaultRepo + "/tag:v1",
			},
		},
		{
			name: "tag",
			desired: Named{
				raw:     "tag:" + defaultTag,
				domain:  defaultDomain,
				repo:    defaultRepo + "/tag",
				tag:     defaultTag,
				repoTag: defaultRepo + "/tag:" + defaultTag,
			},
		},
		{
			name: "xxx.com:5000/abc/tag",
			desired: Named{
				raw:     "xxx.com:5000/abc/tag:" + defaultTag,
				domain:  "xxx.com:5000",
				repo:    "abc/tag",
				tag:     defaultTag,
				repoTag: "abc/tag:" + defaultTag,
			},
		},
	}

	for _, tt := range ts {
		named, err := ParseToNamed(tt.name)
		if err != nil {
			t.Fatalf(err.Error())
		}
		err = compareNamed(named, tt.desired)
		if err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func compareNamed(a, b Named) error {
	type compare struct {
		c, d string
	}
	cs := []compare{{
		c: a.raw,
		d: b.raw,
	}, {
		c: a.tag,
		d: b.tag,
	}, {
		c: a.repoTag,
		d: b.repoTag,
	}, {
		c: a.repo,
		d: b.repo,
	}, {
		c: a.domain,
		d: b.domain,
	}}
	for _, c := range cs {
		if c.d != c.c {
			return fmt.Errorf("%s does not equal to %s", c.c, c.d)
		}
	}
	return nil
	//if reflect.DeepEqual(a, b) {
	//	return nil
	//}
	//return errors.New("not equal")
}

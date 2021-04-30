package registry

import (
	"testing"
)

func TestPingable(t *testing.T) {
	testcases := map[string]struct {
		registry Registry
		expect   bool
	}{
		"Docker": {
			registry: Registry{URL: "https://registry-1.docker.io"},
			expect:   true,
		},
		"GCR_global": {
			registry: Registry{URL: "https://gcr.io"},
			expect:   false,
		},
		"GCR_asia": {
			registry: Registry{URL: "https://asia.gcr.io"},
			expect:   false,
		},
	}
	for label, testcase := range testcases {
		actual := testcase.registry.Pingable()
		if testcase.expect != actual {
			t.Fatalf("%s: expected (%v), got (%v)", label, testcase.expect, actual)
		}
	}
}

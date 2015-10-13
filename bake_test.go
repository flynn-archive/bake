package bake_test

import (
	"testing"

	"github.com/flynn/bake"
)

// Ensure targets can be retrieved by name.
func TestPackage_Target(t *testing.T) {
	pkg := &bake.Package{
		Targets: []*bake.Target{
			{Name: "foo"},
			{Name: "bar"},
			{Name: "baz"},
		},
	}
	if target := pkg.Target("bar"); target == nil || target.Name != "bar" {
		t.Fatalf("unexpected target: %#v", target)
	}
}

// Ensure labels can be parsed into package and target names.
func TestParseLabel(t *testing.T) {
	for i, tt := range []struct {
		in  string
		out bake.Label
	}{
		{"", bake.Label{Package: "", Target: ""}},
		{"foo", bake.Label{Package: "", Target: "foo"}},
		{"foo:bar", bake.Label{Package: "foo", Target: "bar"}},
	} {
		if out := bake.ParseLabel(tt.in); out != tt.out {
			t.Errorf("%d. %s: label:\ngot=%#v\nexp=%#v\n\n", i, tt.in, out, tt.out)
		}
	}
}

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
	if l, err := bake.ParseLabel("foo#test"); err != nil {
		t.Fatal(err)
	} else if l != (bake.Label{Package: "foo", Target: "test"}) {
		t.Fatalf("unexpected label: %#v", l)
	}
}

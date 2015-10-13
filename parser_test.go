package bake_test

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/flynn/bake"
)

// Ensure a package can be parsed from a bakefile.
func TestParser_Parse(t *testing.T) {
	p, err := bake.NewParser().ParseString(`
target("bin/flynn-host", function()
	exec "run1.sh"
	exec "run2.sh"
end)
`)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve target from package.
	target := p.Target("bin/flynn-host")
	if target == nil {
		t.Fatal("expected target")
	} else if !reflect.DeepEqual(target.Commands, []bake.Command{
		&bake.ExecCommand{Text: "run1.sh"},
		&bake.ExecCommand{Text: "run2.sh"},
	}) {
		t.Fatalf("unexpected commands: %s", spew.Sdump(target.Commands))
	}
}

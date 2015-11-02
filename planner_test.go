package bake_test

/*
import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/flynn/bake"
)

// Ensure the planner can plan a single target build with a change on a nested dependency.
func TestPlanner_Plan(t *testing.T) {
	p := bake.NewPlanner(&bake.Package{
		Targets: []*bake.Target{
			{
				Name:   "bin/flynn-blobstore",
				Inputs: []string{"a.go", "b.go"},
			},
			{
				Name:   "build-image",
				Inputs: []string{"bin/flynn-blobstore"},
			},
		},
	})

	p.Plan([]string{"build-image"})
}

// Ensure the planner reuses dependencies that multiple targets depend on.
func TestPlanner_Plan_ReuseTargets(t *testing.T) {
	// B & C both depend on D.
	p := bake.NewPlanner(&bake.Package{
		Targets: []*bake.Target{
			{Name: "A", Inputs: []string{"B", "C"}},
			{Name: "B", Inputs: []string{"D"}},
			{Name: "C", Inputs: []string{"D"}},
			{Name: "D", Inputs: []string{"E"}},
		},
	})

	// Create a plan for when "E" changes. "D" should be reused.
	b, err := p.Plan([]string{"A"})
	if err != nil {
		t.Fatal(err)
	}

	buildA := b.Dependencies()[0]
	buildB, buildC := buildA.Dependencies()[0], buildA.Dependencies()[1]
	if buildB.Dependencies()[0] != buildC.Dependencies()[0] {
		t.Fatalf("mismatched dependencies: %#v != %#v", buildB.Dependencies()[0], buildC.Dependencies()[0])
	}
}

// Ensure the planner returns no build if there are no changes.
func TestPlanner_Plan_NoChange(t *testing.T) {
	p := bake.NewPlanner(&bake.Package{
		Targets: []*bake.Target{
			{
				Name:   "bin/main",
				Inputs: []string{"main.go"},
			},
		},
	})

	// Create a build plan.
	b, err := p.Plan([]string{"bin/main"})
	if err != nil {
		t.Fatal(err)
	} else if b != nil {
		t.Fatalf("unexpected build: %s", spew.Sdump(b))
	}
}

*/

package bake

import (
	"fmt"
)

// Planner represents the object that creates a build plan.
type Planner struct {
	pkg *Package
}

// NewPlanner returns a new instance of Planner.
func NewPlanner(pkg *Package) *Planner {
	return &Planner{pkg: pkg}
}

// Plan creates a build plan for a target in a package.
// The changeset represents a set of files that have changed.
func (p *Planner) Plan(targets []string, changeset map[string]struct{}) (*Build, error) {
	// Create a look up of builds by target so dependencies share references.
	builds := make(map[string]*Build)

	// Default changeset to an empty map.
	if changeset == nil {
		changeset = make(map[string]struct{})
	}

	// Create top-level build.
	b := &Build{}

	for _, target := range targets {
		// Look up target reference.
		t := p.pkg.Target(target)
		if t == nil {
			return nil, fmt.Errorf("target not found: %s", target)
		}

		// Plan build recursively starting from target.
		subbuild, err := p.planTarget(t, changeset, builds)
		if err != nil {
			return nil, err
		} else if subbuild == nil {
			continue
		}
		b.dependencies = append(b.dependencies, subbuild)
	}

	// If there are no subbuilds then return nil.
	if len(b.dependencies) == 0 {
		return nil, nil
	}

	return b, nil
}

func (p *Planner) planTarget(t *Target, changeset map[string]struct{}, builds map[string]*Build) (*Build, error) {
	// Reuse build reference if another target already depends on it.
	if b := builds[t.Name]; b != nil {
		return b, nil
	}

	// Find dependent builds and changed inputs.
	var inputsChanged bool
	var dependencies []*Build
	for _, input := range t.Inputs {
		subtarget := p.pkg.Target(input)

		// If there is no named target then it must be a file.
		// Mark this build as dirty if it's in the changeset.
		if subtarget == nil {
			if _, ok := changeset[input]; ok {
				inputsChanged = true
			}
			continue
		}

		// If input is a target then plan it as a build.
		subbuild, err := p.planTarget(subtarget, changeset, builds)
		if err != nil {
			return nil, err
		} else if subbuild == nil {
			continue
		}
		dependencies = append(dependencies, subbuild)
		inputsChanged = true
	}

	// If no input files or targets have changed then ignore build.
	if !inputsChanged {
		return nil, nil
	}

	// Create build and add it to the lookup.
	b := &Build{
		target:       t,
		dependencies: dependencies,
	}
	builds[b.target.Name] = b

	return b, nil
}

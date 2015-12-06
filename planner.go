package bake

// Planner represents the object that creates a build plan.
// This type is not safe for multiple goroutines.
type Planner struct {
	pkg *Package

	builds map[string]*Build

	// Used to determine dirty targets since last build.
	Snapshot *Snapshot
}

// NewPlanner returns a new instance of Planner.
func NewPlanner(pkg *Package) *Planner {
	return &Planner{pkg: pkg}
}

// Plan creates a build plan for a target in a package.
func (p *Planner) Plan(patterns []string) (*Build, error) {
	// Create a lookup of builds by target so dependencies share references.
	p.builds = make(map[string]*Build)
	defer func() { p.builds = nil }()

	dependencies, err := p.planMatches(patterns)
	if err != nil {
		return nil, err
	}

	b := newBuild(nil)
	b.dependencies = dependencies
	return b, nil
}

// planMatches returns builds for all targets matching any of the patterns.
func (p *Planner) planMatches(patterns []string) ([]*Build, error) {
	// Plan each pattern.
	var subbuilds []*Build
	for _, pattern := range patterns {
		a, err := p.planMatch(pattern)
		if err != nil {
			return nil, err
		}
		subbuilds = append(subbuilds, a...)
	}
	return Builds(subbuilds).dedupe(), nil
}

// planMatch plans all targets matching a pattern.
func (p *Planner) planMatch(pattern string) ([]*Build, error) {
	targets, err := p.pkg.MatchTargets(pattern)
	if err != nil {
		return nil, err
	}

	var builds []*Build
	for _, t := range targets {
		b, err := p.planTarget(t)
		if err != nil {
			return nil, err
		} else if b == nil {
			continue
		}
		builds = append(builds, b)
	}
	return builds, nil
}

// planTarget plans a single target.
func (p *Planner) planTarget(t *Target) (*Build, error) {
	// Reuse build reference if another target already depends on it.
	if b := p.builds[t.Name]; b != nil {
		return b, nil
	}

	// Find dependent builds and changed inputs.
	dependencies, err := p.planMatches(t.Dependencies)
	if err != nil {
		return nil, err
	}

	// If there are no dependencies then check if target changed or its files are dirty.
	if len(dependencies) == 0 && p.Snapshot != nil {
		if dirty, err := p.Snapshot.IsTargetDirty(t); err != nil {
			return nil, err
		} else if !dirty {
			return nil, nil
		}
	}

	// Create build and add it to the lookup.
	b := newBuild(t)
	b.dependencies = dependencies

	// Add it it to the lookup.
	p.builds[b.target.Name] = b

	return b, nil
}

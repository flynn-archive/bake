package bake

import (
	"net/url"
	"path"
)

// Package represents a collection of targets.
type Package struct {
	Name    string
	Targets []*Target
}

// Target returns a target by name or by output.
func (p *Package) Target(name string) *Target {
	for _, t := range p.Targets {
		if t.Name == name {
			return t
		}

		for _, output := range t.Outputs {
			if output == name {
				return t
			}
		}
	}

	return nil
}

// Target represents a rule or file within a package.
type Target struct {
	Name    string   // e.g. "test"
	Command string   // command to execute
	Inputs  []string // dependent input files
	Outputs []string // declared outputs
}

type Label struct {
	Package string
	Target  string
}

// ParseLabel parses a URI string into a label.
func ParseLabel(s string) (Label, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Label{}, err
	}

	// Join host and path together to form package.
	l := Label{}
	if u.Host != "" && u.Path != "" {
		l.Package = path.Join(u.Host, u.Path)
	} else if u.Host != "" {
		l.Package = u.Host
	} else if u.Path != "" {
		l.Package = u.Path
	}

	// Extract target from fragment. Otherwise use base of package.
	if u.Fragment != "" {
		l.Target = u.Fragment
	} else {
		l.Target = path.Base(l.Package)
	}

	return l, nil
}

package bake

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Shopify/go-lua"
	"github.com/flynn/bake/assets"
)

//go:generate go-bindata -ignore=assets/assets.go -o assets/assets.go -pkg assets -prefix assets/ assets

// Parser represents a recursive directory parser.
type Parser struct {
	state  *lua.State
	target *Target // current target being built

	base string // root directory
	path string // working directory passed to targets

	// Package being built by the parser.
	// It is incrementally added to on every parse.
	Package *Package
}

// NewParser returns a new instance of Parser.
func NewParser() *Parser {
	p := &Parser{
		state:   lua.NewState(),
		Package: &Package{},
	}
	p.init()
	return p
}

// ParseDir recursively parses all bakefiles in a directory tree.
// If a directory contains a Bakefile then it is parsed and added to the package.
// All targets and their names are relative from path.
func (p *Parser) ParseDir(path string) error {
	return p.parseDir(path, "")
}

func (p *Parser) parseDir(base, path string) error {
	// Parse Bakefile in this directory first.
	if err := p.parseFile(base, filepath.Join(path, "Bakefile")); os.IsNotExist(err) {
		// nop
	} else if err != nil {
		return err
	}

	// Read all files in path.
	fis, err := readdir(filepath.Join(base, path))
	if err != nil {
		return err
	}

	// Recursively parse each directory.
	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}
		if err := p.parseDir(base, filepath.Join(path, fi.Name())); err != nil {
			return err
		}
	}

	return nil
}

// parseFile parses a file at path.
func (p *Parser) parseFile(base, path string) error {
	// Open file for reading.
	f, err := os.Open(filepath.Join(base, path))
	if err != nil {
		return err
	}
	defer f.Close()

	// Set the current working directory for all targets created.
	// Reset after parsing file or on error.
	p.base, p.path = base, filepath.Dir(path)
	defer func() { p.base, p.path = "", "" }()

	// Load script into state.
	if err := p.state.Load(f, path, ""); err != nil {
		return err
	}

	// Execute script.
	if err := p.state.ProtectedCall(0, 0, 0); err != nil {
		return err
	}

	return nil
}

func (p *Parser) init() {
	// Import standard library.
	lua.OpenLibraries(p.state)

	// Load bake libraries.
	for _, name := range []string{
		"shim.lua", // intermediate layer to go-lua
		"bake.lua",
		"docker.lua",
		"git.lua",
		"go.lua",
	} {
		if err := lua.LoadBuffer(p.state, string(assets.MustAsset(name)), name, ""); err != nil {
			panic(err)
		}
		p.state.Call(0, 0)
	}

	// Add built-in functions.
	p.state.Register("__bake_begin_target", p.beginTarget)
	p.state.Register("__bake_end_target", p.endTarget)
	p.state.Register("__bake_set_title", p.setTitle)
	p.state.Register("exec", p.exec)
	p.state.Register("depends", p.depends)
}

// beginTarget initializes a target on the package.
func (p *Parser) beginTarget(l *lua.State) int {
	name := lua.CheckString(l, 1)
	dependencies, _ := l.ToUserData(2).(luaDependencies)

	// Mark as "phony" if it starts with an at-sign.
	var phony bool
	if strings.HasPrefix(name, "@") {
		name = strings.TrimPrefix(name, "@")
		phony = true
	}

	p.target = &Target{
		Name:    path.Join(p.path, name),
		Phony:   phony,
		WorkDir: p.path,
		Inputs:  make([]string, len(dependencies)),
	}

	// Copy dependencies with prepended path.
	for i := range dependencies {
		p.target.Inputs[i] = path.Join(p.path, dependencies[i])
	}

	return 0
}

// endTarget finalizes the current target and adds it to the package.
func (p *Parser) endTarget(l *lua.State) int {
	p.Package.Targets = append(p.Package.Targets, p.target)
	p.target = nil
	return 0
}

// exec appends an "exec" command to the current target.
func (p *Parser) exec(l *lua.State) int {
	cmd := &ExecCommand{}
	for i, n := 1, l.Top(); i <= n; i++ {
		cmd.Args = append(cmd.Args, lua.CheckString(l, i))
	}

	p.target.Commands = append(p.target.Commands, cmd)

	return 0
}

// setTitle sets the title shown to users for the current target.
func (p *Parser) setTitle(l *lua.State) int {
	p.target.Title = lua.CheckString(l, 1)
	return 0
}

// depends returns a list of strings as dependencies.
func (p *Parser) depends(l *lua.State) int {
	dependencies := make(luaDependencies, 0)
	for i, n := 1, l.Top(); i <= n; i++ {
		dependencies = append(dependencies, lua.CheckString(l, i))
	}

	l.PushUserData(dependencies)
	return 1
}

// luaDependencies represents a list of dependency names.
type luaDependencies []string

// readdir returns a slice of all files in path.
func readdir(path string) ([]os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Readdir(0)
}

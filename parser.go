package bake

import (
	"io"
	"strings"

	"github.com/Shopify/go-lua"
	"github.com/flynn/bake/assets"
)

//go:generate go-bindata -ignore=assets/assets.go -o assets/assets.go -pkg assets -prefix assets/ assets

// Parser represents a parser for a Bakefile.
type Parser struct {
}

// NewParser returns a new instance of Parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a package definition from r.
func (p *Parser) Parse(r io.Reader) (*Package, error) {
	// Initialize new Lua state.
	l := newLuaState()

	// Load script into state.
	if err := l.Load(r, "Bakefile", ""); err != nil {
		return nil, err
	}

	// Execute script.
	if err := l.ProtectedCall(0, 0, 0); err != nil {
		return nil, err
	}

	return l.pkg, nil
}

// ParseString parses a package definition from s.
func (p *Parser) ParseString(s string) (*Package, error) {
	return p.Parse(strings.NewReader(s))
}

// luaState represents a wrapper around the Lua state object.
// It holds a reference to the package currently being built.
type luaState struct {
	*lua.State
	pkg *Package

	target *Target
}

// newLuaState returns a new instance of luaState.
func newLuaState() *luaState {
	l := &luaState{
		State: lua.NewState(),
		pkg:   &Package{},
	}

	// Import standard library.
	lua.OpenLibraries(l.State)

	// Load bake libraries.
	for _, name := range []string{
		"shim.lua", // intermediate layer to go-lua
		"bake.lua",
		"docker.lua",
		"git.lua",
		"go.lua",
	} {
		if err := lua.LoadBuffer(l.State, string(assets.MustAsset(name)), name, ""); err != nil {
			panic(err)
		}
		l.Call(0, 0)
	}

	// Add built-in functions.
	l.Register("__bake_begin_target", l.beginTarget)
	l.Register("__bake_end_target", l.endTarget)
	l.Register("__bake_set_title", l.setTitle)
	l.Register("exec", l.exec)
	l.Register("depends", l.depends)

	return l
}

// beginTarget initializes a target on the package.
func (l *luaState) beginTarget(_ *lua.State) int {
	name := lua.CheckString(l.State, 1)
	dependencies, _ := l.State.ToUserData(2).(luaDependencies)

	// Mark as "phony" if it starts with an at-sign.
	var phony bool
	if strings.HasPrefix(name, "@") {
		name = strings.TrimPrefix(name, "@")
		phony = true
	}

	l.target = &Target{
		Name:   name,
		Phony:  phony,
		Inputs: []string(dependencies),
	}

	return 0
}

// endTarget finalizes the current target and adds it to the package.
func (l *luaState) endTarget(_ *lua.State) int {
	l.pkg.Targets = append(l.pkg.Targets, l.target)
	l.target = nil
	return 0
}

// exec appends an "exec" command to the current target.
func (l *luaState) exec(_ *lua.State) int {
	cmd := &ExecCommand{}
	for i, n := 1, l.Top(); i <= n; i++ {
		cmd.Args = append(cmd.Args, lua.CheckString(l.State, i))
	}

	l.target.Commands = append(l.target.Commands, cmd)

	return 0
}

// setTitle sets the title shown to users for the current target.
func (l *luaState) setTitle(_ *lua.State) int {
	l.target.Title = lua.CheckString(l.State, 1)
	return 0
}

// depends returns a list of strings as dependencies.
func (l *luaState) depends(_ *lua.State) int {
	dependencies := make(luaDependencies, 0)
	for i, n := 1, l.Top(); i <= n; i++ {
		dependencies = append(dependencies, lua.CheckString(l.State, i))
	}

	l.PushUserData(dependencies)
	return 1
}

// luaDependencies represents a list of dependency names.
type luaDependencies []string

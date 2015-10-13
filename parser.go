package bake

import (
	"io"
	"strings"

	"github.com/Shopify/go-lua"
)

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
	if err := l.Load(r, "main", ""); err != nil {
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

	// Load shim.
	if err := lua.LoadBuffer(l.State, shim, "shim", ""); err != nil {
		panic(err)
	}
	l.Call(0, 0)

	// Add built-in bake functions.
	l.Register("__bake_begin_target", l.beginTarget)
	l.Register("__bake_end_target", l.endTarget)
	l.Register("__bake_add_command", l.addCommand)

	return l
}

// beginTarget initializes a target on the package.
func (l *luaState) beginTarget(_ *lua.State) int {
	name := lua.CheckString(l.State, 1)
	l.target = &Target{Name: name}
	return 0
}

// endTarget finalizes the current target and adds it to the package.
func (l *luaState) endTarget(_ *lua.State) int {
	l.pkg.Targets = append(l.pkg.Targets, l.target)
	l.target = nil
	return 0
}

// addCommand appends a command to the current target.
func (l *luaState) addCommand(_ *lua.State) int {
	name := lua.CheckString(l.State, 1)

	// Command type and arg list determined by name.
	var cmd Command
	switch name {
	case "exec":
		cmd = &ExecCommand{
			Text: lua.CheckString(l.State, 2),
		}
	default:
		panic("invalid command type: " + name)
	}

	// Add to the list of commands.
	l.target.Commands = append(l.target.Commands, cmd)
	return 0
}

// shim provides an intermediate layer to the parser.
//
// This is required because the go-lua library does not allow us to save and
// execute closure references from Go so we need to have Lua do this for us.
const shim = `
function target(name, fn)
	__bake_begin_target(name)
	fn()
	__bake_end_target(name)
end

function exec(text)
	__bake_add_command("exec", text)
end
`

// Copyright 2014 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package cmdset implements multi-cmd command-line parsing.

	Multi-cmd command-line format:
		program command [arguments]

	For example, the command-line of "go" is this type, its usage is
		go command [arguments]
	The command can be build, clean, fmt, etc.

	Usage:

	Define a new cmd using cmdset.NewCmd() or cmdset.NewCmdVar().
	The Cmd structure returned contains flag.FlagSet, you can use
	it to define flags for the new cmd. For information	about
	how to define flags, see the documentation for flag.

	After all cmds are defined, call
		cmdset.Parse()
	to parse the command line into one of the defined cmds.
	The	selected cmd is called winning cmd, you can obtain it by
	calling cmdset.Winning().

	After parsing, the arguments after the cmd are available as
	the	slice cmdset.Winning().Args() or individually as
	cmdset.Winning().Arg(i). The arguments are indexed from 0
	through cmdset.Winning().NArg()-1.
*/
package cmdset

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

// ErrHelp is the error returned if the cmd help is invoked but no such cmd is defined.
var ErrHelp = errors.New("cmd: help requested")

// ErrorHandling defines how to handle cmd parsing errors.
type ErrorHandling int

const (
	ContinueOnError ErrorHandling = iota
	ExitOnError
	PanicOnError
)

// A CmdSet represents a set of defined cmds. The zero value of a CmdSet
// is invalid, please use NewCmdSet() or call CmdSet.Init().
type CmdSet struct {
	// CustomHelp is the function called when an error occurs while parsing.
	// The field is a function (not a method) that may be changed to point to
	// a custom error handler.
	CustomHelp func()

	name          string
	parsed        bool
	winning       string
	cmds          map[string]*Cmd
	errorHandling ErrorHandling
	output        io.Writer
	maxCmdLen     int
}

// A Cmd represents the state of a cmd.
type Cmd struct {
	Name         string // name as it appears on command line
	Explain      string // explain message
	flag.FlagSet        // the flags of the cmd
}

func (cmd *Cmd) Help() {
	cmd.Parse([]string{"-help"})
}

// NewCmdVar defines a cmd with specified name, and explain string.
// The argument cmd points to a Cmd variable in which to store the flags of the cmd.
func (c *CmdSet) NewCmdVar(cmd *Cmd, name string, explain string) {
	cmd.Name = name
	cmd.Explain = explain
	cmd.Init(name, flag.ErrorHandling(c.errorHandling))
	cmd.SetOutput(c.output)
	c.cmds[name] = cmd
	if len(name) > c.maxCmdLen {
		c.maxCmdLen = len(name)
	}
}

// NewCmdVar defines a cmd with specified name, and explain string.
// The argument cmd points to a Cmd variable in which to store the flags of the cmd.
func NewCmdVar(cmd *Cmd, name string, explain string) {
	CommandLine.NewCmdVar(cmd, name, explain)
}

// NewCmd defines a cmd with specified name, and explain string.
// The return value is the address of a Cmd variable in which to store the flags of the cmd.
func (c *CmdSet) NewCmd(name string, explain string) *Cmd {
	cmd := new(Cmd)
	c.NewCmdVar(cmd, name, explain)
	return cmd
}

// NewCmd defines a cmd with specified name, and explain string.
// The return value is the address of a Cmd variable in which to store the flags of the cmd.
func NewCmd(name string, explain string) *Cmd {
	return CommandLine.NewCmd(name, explain)
}

// sortCmds returns the cmds as a slice in lexicographical sorted order.
func sortCmds(cmds map[string]*Cmd) []*Cmd {
	list := make(sort.StringSlice, len(cmds))
	i := 0
	for n := range cmds {
		list[i] = n
		i++
	}
	list.Sort()
	result := make([]*Cmd, len(list))
	for i, name := range list {
		result[i] = cmds[name]
	}
	return result
}

// SetOutput sets the destination for help and error messages.
// If output is nil, os.Stderr is used.
func (c *CmdSet) SetOutput(output io.Writer) {
	if output == nil {
		output = os.Stderr
	}
	c.output = output
	for _, cmd := range c.cmds {
		cmd.SetOutput(output)
	}
}

// SetOutput sets the destination for help and error messages.
// If output is nil, os.Stderr is used.
func SetOutput(output io.Writer) {
	CommandLine.SetOutput(output)
}

// Winning returns the winning Cmd structure.
func (c *CmdSet) Winning() *Cmd {
	return c.Lookup(c.winning)
}

// Winning returns the winning Cmd structure.
func Winning() *Cmd {
	return CommandLine.Winning()
}

// Lookup returns the Cmd structure of the named cmd, returning nil if none exists.
func (c *CmdSet) Lookup(name string) *Cmd {
	return c.cmds[name]
}

// Lookup returns the Cmd structure of the named cmd, returning nil if none exists.
func Lookup(name string) *Cmd {
	return CommandLine.Lookup(name)
}

// Visit visits the cmds in lexicographical order, calling fn for each.
func (c *CmdSet) Visit(fn func(*Cmd)) {
	for _, cmd := range sortCmds(c.cmds) {
		fn(cmd)
	}
}

// Visit visits the cmds in lexicographical order, calling fn for each.
func Visit(fn func(*Cmd)) {
	CommandLine.Visit(fn)
}

func (c *CmdSet) failf(format string, a ...interface{}) error {
	err := fmt.Errorf(format, a...)
	fmt.Fprintln(c.output, err)
	c.help()
	return err
}

func (c *CmdSet) help() {
	if c.CustomHelp == nil {
		c.defaultHelp()
	} else {
		c.CustomHelp()
	}
}

func (c *CmdSet) defaultHelp() {
	fmt.Fprintln(c.output)
	fmt.Fprintln(c.output, "Usage:")
	fmt.Fprintln(c.output)
	fmt.Fprintf(c.output, "%s command [arguments]\n", c.name)
	fmt.Fprintln(c.output)
	fmt.Fprintln(c.output, "The commands are:")
	fmt.Fprintln(c.output)
	c.Visit(func(cmd *Cmd) {
		n := cmd.Name
		if len(n) < c.maxCmdLen {
			b := make([]byte, c.maxCmdLen-len(n))
			for i := range b {
				b[i] = ' '
			}
			n = n + string(b)
		}
		fmt.Fprintf(c.output, "    %s  %s\n", n, cmd.Explain)
	})
	fmt.Fprintln(c.output)
	fmt.Fprintf(c.output, "Use \"%s help [command]\" for more information about a command.", c.name)
	fmt.Fprintln(c.output)
	fmt.Fprintln(c.output)
}

func (c *CmdSet) parseCmd(arguments []string) error {
	if len(arguments) == 0 {
		c.help()
		return ErrHelp
	}

	name := arguments[0]

	cmd, alreadythere := c.cmds[name]
	if !alreadythere {
		// special case for nice help message.
		// CmdSet help
		if name == "-h" || name == "-help" || name == "--help" {
			c.help()
			return ErrHelp
		}
		if name == "help" {
			if len(arguments) == 1 {
				c.help()
				return ErrHelp
			}
			name2 := arguments[1]
			cmd2, alreadythere2 := c.cmds[name2]
			if !alreadythere2 {
				c.help()
				return ErrHelp
			}
			// Cmd Help
			cmd2.Help()
			return ErrHelp
		}

		return c.failf("unknown cmd: %s", name)
	}

	c.winning = name
	return cmd.Parse(arguments[1:])
}

// Parse parses cmd definitions from the argument list, the first argument
// is the cmd name. Must be called after all cmds in the CmdSet
// are defined and before winning are accessed by the program.
func (c *CmdSet) Parse(arguments []string) error {
	c.parsed = true
	err := c.parseCmd(arguments)
	if err != nil {
		switch c.errorHandling {
		case ContinueOnError:
			return err
		case ExitOnError:
			os.Exit(2)
		case PanicOnError:
			panic(err)
		}
	}
	return nil
}

// Parse parses cmd definitions from the command-line (os.Args[1:]). Must be called
// after all cmds are defined and before winning are accessed by the program.
func Parse() {
	// Ignore errors; CommandLine is set for ExitOnError.
	CommandLine.Parse(os.Args[1:])
}

// Parsed reports whether c.Parse has been called.
func (c *CmdSet) Parsed() bool {
	return c.parsed
}

// Parsed returns true if the command-line have been parsed.
func Parsed() bool {
	return CommandLine.Parsed()
}

// Help prints a help message.
func (c *CmdSet) Help() {
	c.help()
}

// Help prints a help message.
func Help() {
	CommandLine.Help()
}

// CommandLine is the default CmdSet of command-line, parsed from os.Args.
// The top-level functions such as NewCmd, Winning, Lookup, and on are wrappers for the
// methods of CommandLine.
var CommandLine = NewCmdSet(os.Args[0], ExitOnError)

// NewCmdSet returns a new, empty cmd set with the specified name and
// error handling property.
func NewCmdSet(name string, errorHandling ErrorHandling) *CmdSet {
	c := new(CmdSet)
	c.Init(name, errorHandling)
	return c
}

// Init sets the name and error handling property for a cmd set.
func (c *CmdSet) Init(name string, errorHandling ErrorHandling) {
	c.name = name
	c.errorHandling = errorHandling
	c.cmds = make(map[string]*Cmd)
	c.output = os.Stderr
	c.maxCmdLen = 10
}

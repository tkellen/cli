package cli

import "fmt"

// Tree describes a command tree.
type Tree struct {
	// Fn must be type func([]string) error, Fn or Tree
	Fn          interface{}
	SubCommands Map
}

type Fn struct {
	Fn      func([]string) error
	MinArgs int
	Help    func([]string) error
}

// Map holds command names that map to either func([]string) error, Fn or Tree.
type Map map[string]interface{}

// Dispatch traverses the tree of commands and executes the correct one.
func (t Tree) Dispatch(args []string) error {
	argsUntilSub := 0
	if fn, ok := t.Fn.(Fn); ok {
		argsUntilSub = fn.MinArgs
	}
	if len(args) > argsUntilSub && t.SubCommands != nil {
		subCommand := args[argsUntilSub]
		for key, item := range t.SubCommands {
			if subCommand == key {
				args = append(args[0:argsUntilSub], args[argsUntilSub+1:]...)
				return run(item, args)
			}
		}
	}
	return run(t.Fn, args)
}

func run(input interface{}, args []string) error {
	if tree, ok := input.(Tree); ok {
		return tree.Dispatch(args)
	}
	if cmd, ok := input.(Fn); ok {
		return minArgs(cmd.MinArgs, cmd.Fn, cmd.Help)(args)
	}
	if cmd, ok := input.(func([]string) error); ok {
		return cmd(args)
	}
	return Invalid(args)
}

// Invalid is the function that will be executed run cannot find the right
// type for its input interface.
func Invalid(args []string) error {
	return fmt.Errorf("invalid interface for: %s", args)
}

// minArgs wraps a command and runs a fallback when the command is executed
// if it doesn't satisfy a minimum number of arguments.
func minArgs(min int, fn func([]string) error, fallback func([]string) error) func([]string) error {
	return func(args []string) error {
		if min == 0 || len(args) < min {
			return fallback(args)
		}
		return fn(args)
	}
}

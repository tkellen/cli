package cli_test

import (
	"errors"
	"fmt"
	"github.com/tkellen/cli"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func ioEOF(args []string) error {
	return fmt.Errorf("%w: %s", io.EOF, args)
}
func osErrNotExist(args []string) error {
	return fmt.Errorf("%w: %s", os.ErrNotExist, args)
}
func osErrExist(args []string) error {
	return fmt.Errorf("%w: %s", os.ErrExist, args)
}
func osErrClosed(args []string) error {
	return fmt.Errorf("%w: %s", os.ErrClosed, args)
}
func httpErrContentLength(args []string) error {
	return fmt.Errorf("%w: %s", http.ErrContentLength, args)
}

// TODO: test min args and args reordering stuff
func TestDispatchCorrectCommand(t *testing.T) {
	tree := cli.Tree{
		Fn: ioEOF,
		SubCommands: cli.Map{
			"os": cli.Tree{
				Fn: osErrNotExist,
				SubCommands: cli.Map{
					"exist":  osErrExist,
					"closed": osErrClosed,
				},
			},
			"http": httpErrContentLength,
			"io":   ioEOF,
			"bad":  "citizen",
		},
	}
	table := map[string]struct {
		fn   interface{}
		args []string
	}{
		"": {
			fn:   tree.Fn,
			args: []string{},
		},
		"foo": {
			fn:   tree.Fn,
			args: []string{"foo"},
		},
		"os exist": {
			fn:   tree.SubCommands["os"].(cli.Tree).SubCommands["exist"].(func([]string) error),
			args: []string{},
		},
		"os exist foo bar baz": {
			fn:   tree.SubCommands["os"].(cli.Tree).SubCommands["exist"].(func([]string) error),
			args: []string{"foo", "bar", "baz"},
		},
		"os closed !": {
			fn:   tree.SubCommands["os"].(cli.Tree).SubCommands["closed"].(func([]string) error),
			args: []string{"!"},
		},
		"os beep boop": {
			fn:   tree.SubCommands["os"].(cli.Tree).Fn,
			args: []string{"beep", "boop"},
		},
	}
	for name, test := range table {
		test := test
		t.Run(name, func(t *testing.T) {
			actual := tree.Dispatch(strings.Fields(name))
			expected := test.fn.(func([]string) error)(test.args)
			if errors.Is(actual, expected) {
				t.Fatalf("expected error %s, got %s", expected, actual)
			}
			if expected.Error() != actual.Error() {
				t.Fatalf("expected fn/args of %s, got %s", expected, actual)
			}
		})
	}
	if !strings.Contains(tree.Dispatch([]string{"bad", "times"}).Error(), "invalid interface") {
		t.Fatal("expected Invalid function to be called on bad interface")
	}
}

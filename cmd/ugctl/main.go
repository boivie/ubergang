// ugctl is a command-line client for Ubergang's public API.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Set by the release build, see .goreleaser.yaml.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// Exit codes. They are part of the CLI's interface, documented in
// docs/cli.md.
const (
	exitOK           = 0
	exitError        = 1
	exitUsage        = 2
	exitNotFound     = 3
	exitConflict     = 4
	exitUnauthorized = 5
)

type app struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	// Global flags.
	url        string
	configPath string
	output     string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	a := &app{stdin: stdin, stdout: stdout, stderr: stderr, output: outputText}
	root := newRootCmd(a)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return exitOK
	}
	a.printError(err)
	return exitCode(err)
}

func newRootCmd(a *app) *cobra.Command {
	root := groupCmd(&cobra.Command{
		Use:   "ugctl",
		Short: "Control an Ubergang server",
		Long: `ugctl controls an Ubergang server through its public API.

Authenticate with an API key, created under Users → API keys in the web UI,
either with "ugctl login" or with the UBERGANG_URL and UBERGANG_API_KEY
environment variables.`,
		Version: versionString(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if a.output != outputText && a.output != outputJSON {
				return usageErrorf("invalid --output %q: must be %q or %q", a.output, outputText, outputJSON)
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	},
		newBackupCmd(a),
		newLoginCmd(a),
		newLogoutCmd(a),
		newMqttCmd(a),
		newVersionCmd(a),
	)
	root.SetIn(a.stdin)
	root.SetOut(a.stdout)
	root.SetErr(a.stderr)
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return &usageError{err}
	})
	root.CompletionOptions.HiddenDefaultCmd = true

	flags := root.PersistentFlags()
	flags.StringVar(&a.url, "url", "",
		"URL of the server's admin host, such as https://admin.example.com (env UBERGANG_URL)")
	flags.StringVar(&a.configPath, "config", "",
		"config file (env UBERGANG_CONFIG, default ~/.config/ubergang/ugctl.yaml)")
	flags.StringVarP(&a.output, "output", "o", outputText, "output format: text or json")
	return root
}

// groupCmd sets up a command that only has subcommands. Unlike cobra's
// default, an unknown subcommand is an error rather than showing the help.
func groupCmd(cmd *cobra.Command, children ...*cobra.Command) *cobra.Command {
	cmd.Args = cobra.ArbitraryArgs
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return usageErrorf("unknown command %q for %q", args[0], cmd.CommandPath())
		}
		return cmd.Help()
	}
	cmd.AddCommand(children...)
	return cmd
}

// exactArgs is cobra.ExactArgs, but returns a usage error.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			return &usageError{err}
		}
		return nil
	}
}

type usageError struct{ err error }

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

func usageErrorf(format string, args ...any) error {
	return &usageError{fmt.Errorf(format, args...)}
}

func exitCode(err error) int {
	var usageErr *usageError
	if errors.As(err, &usageErr) {
		return exitUsage
	}
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		switch apiErr.code {
		case "not_found":
			return exitNotFound
		case "conflict":
			return exitConflict
		case "unauthorized", "insufficient_scope":
			return exitUnauthorized
		}
	}
	return exitError
}

func newVersionCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, d := buildInfo()
			if a.output == outputJSON {
				return a.printJSON(map[string]string{"version": version, "commit": c, "date": d})
			}
			_, err := fmt.Fprintln(a.stdout, "ugctl "+versionString())
			return err
		},
	}
}

// buildInfo returns the commit and date, falling back to what the Go
// toolchain recorded for builds that don't set them, such as "go build".
func buildInfo() (string, string) {
	c, d := commit, date
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch {
			case s.Key == "vcs.revision" && c == "":
				c = s.Value
			case s.Key == "vcs.time" && d == "":
				d = s.Value
			}
		}
	}
	return c, d
}

func versionString() string {
	c, d := buildInfo()
	s := version
	if c != "" {
		if len(c) > 12 {
			c = c[:12]
		}
		s += " (" + c
		if d != "" {
			s += ", " + d
		}
		s += ")"
	}
	return s
}

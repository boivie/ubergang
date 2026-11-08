package main

import (
	"boivie/ubergang/server/api"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

const (
	outputText = "text"
	outputJSON = "json"
)

// printJSON writes `v` as indented JSON to stdout.
func (a *app) printJSON(v any) error {
	return writeJSON(a.stdout, v)
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// printError writes `err` to stderr: as the public API's error object for
// JSON output, so that callers can always parse it the same way, or as a
// line of text otherwise.
func (a *app) printError(err error) {
	code, message := "error", err.Error()
	var apiErr *apiError
	var usageErr *usageError
	switch {
	case errors.As(err, &apiErr):
		code, message = apiErr.code, apiErr.message
	case errors.As(err, &usageErr):
		code = "usage"
	}
	if a.output == outputJSON {
		_ = writeJSON(a.stderr, api.PublicError{
			Error: api.PublicErrorDetails{Code: code, Message: message},
		})
		return
	}
	if apiErr != nil {
		message = fmt.Sprintf("%s (%s)", message, code)
	}
	_, _ = fmt.Fprintln(a.stderr, "Error: "+message)
	if usageErr != nil {
		_, _ = fmt.Fprintln(a.stderr, "Run 'ugctl --help' for usage.")
	}
}

// printTable writes a table with a header row to stdout.
func (a *app) printTable(header []string, rows [][]string) error {
	tw := tabwriter.NewWriter(a.stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, row := range rows {
		_, _ = fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

// field is a labeled value in printFields. A value with several lines is
// printed with each line aligned under the first.
type field struct {
	label string
	lines []string
}

func (a *app) printFields(fields []field) error {
	tw := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	for _, f := range fields {
		lines := f.lines
		if len(lines) == 0 {
			lines = []string{"-"}
		}
		for i, line := range lines {
			label := ""
			if i == 0 {
				label = f.label + ":"
			}
			_, _ = fmt.Fprintf(tw, "%s\t%s\n", label, line)
		}
	}
	return tw.Flush()
}

func (a *app) printMessage(format string, args ...any) error {
	_, err := fmt.Fprintf(a.stdout, format+"\n", args...)
	return err
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

// formatValues returns the values as sorted "key=value" strings.
func formatValues(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for k, v := range values {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

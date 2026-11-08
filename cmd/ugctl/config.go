package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

const (
	envURL    = "UBERGANG_URL"
	envAPIKey = "UBERGANG_API_KEY"
	envConfig = "UBERGANG_CONFIG"
)

// config is the config file, written by "ugctl login".
type config struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

func (a *app) resolveConfigPath() (string, error) {
	if a.configPath != "" {
		return a.configPath, nil
	}
	if p := os.Getenv(envConfig); p != "" {
		return p, nil
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "ubergang", "ugctl.yaml"), nil
	}
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "ubergang", "ugctl.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ubergang", "ugctl.yaml"), nil
}

// readConfig reads the config file. A missing file is an empty config.
func (a *app) readConfig() (*config, error) {
	path, err := a.resolveConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &config{}, nil
	} else if err != nil {
		return nil, err
	}
	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file %s: %w", path, err)
	}
	return &cfg, nil
}

func (a *app) writeConfig(cfg *config) (string, error) {
	path, err := a.resolveConfigPath()
	if err != nil {
		return "", err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	// Write to a temporary file first, so that a failed write doesn't lose
	// the existing config, and so that the key is never readable by others.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ugctl-*.yaml")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := tmp.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		_ = tmp.Close()
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return path, os.Rename(tmp.Name(), path)
}

// resolveURL returns the server URL from, in order: the --url flag, the
// environment and the config file.
func (a *app) resolveURL(cfg *config) (string, error) {
	raw := a.url
	if raw == "" {
		raw = os.Getenv(envURL)
	}
	if raw == "" {
		raw = cfg.URL
	}
	if raw == "" {
		return "", fmt.Errorf("no server URL: run \"ugctl login <url>\", or set %s and %s", envURL, envAPIKey)
	}
	return normalizeURL(raw)
}

// newClient returns a client for the configured server. The API key from the
// config file is only used for the server it was saved for, so that pointing
// ugctl at another server with --url never sends it there.
func (a *app) newClient() (*client, error) {
	cfg, err := a.readConfig()
	if err != nil {
		return nil, err
	}
	baseURL, err := a.resolveURL(cfg)
	if err != nil {
		return nil, err
	}
	apiKey := os.Getenv(envAPIKey)
	if apiKey == "" && cfg.APIKey != "" {
		if savedURL, err := normalizeURL(cfg.URL); err == nil && savedURL == baseURL {
			apiKey = cfg.APIKey
		}
	}
	if apiKey == "" {
		return nil, fmt.Errorf("no API key for %s: run \"ugctl login %s\", or set %s", baseURL, baseURL, envAPIKey)
	}
	return newClient(baseURL, apiKey), nil
}

func newLoginCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "login [url]",
		Short: "Save the server URL and an API key",
		Long: `Saves the URL of the server's admin host and an API key to the config file,
after checking that the key works.

The API key is read from the terminal, or from stdin if it isn't a terminal:

  ugctl login https://admin.example.com
  ugctl login https://admin.example.com < key.txt

Alternatively, set the UBERGANG_URL and UBERGANG_API_KEY environment
variables, which take precedence over the config file.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return usageErrorf("accepts at most 1 arg, received %d", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := a.readConfig()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				a.url = args[0]
			}
			baseURL, err := a.resolveURL(cfg)
			if err != nil {
				return err
			}
			apiKey, err := a.readSecret("API key for " + baseURL + ": ")
			if err != nil {
				return err
			}
			apiKey = strings.TrimSpace(apiKey)
			if apiKey == "" {
				return errors.New("no API key given")
			}
			// Every key has at least the read scope.
			if _, err := newClient(baseURL, apiKey).listMqttProfiles(cmd.Context()); err != nil {
				return err
			}
			path, err := a.writeConfig(&config{URL: baseURL, APIKey: apiKey})
			if err != nil {
				return err
			}
			if a.output == outputJSON {
				return a.printJSON(map[string]string{"url": baseURL, "config": path})
			}
			return a.printMessage("Logged in to %s. Saved to %s.", baseURL, path)
		},
	}
}

func newLogoutCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved server URL and API key",
		Long: `Removes the config file written by "ugctl login". This doesn't revoke the
API key; delete it in the web UI to do that.`,
		Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.resolveConfigPath()
			if err != nil {
				return err
			}
			err = os.Remove(path)
			removed := err == nil
			if errors.Is(err, fs.ErrNotExist) {
				err = nil
			}
			if err != nil {
				return err
			}
			if a.output == outputJSON {
				return a.printJSON(map[string]any{"config": path, "removed": removed})
			}
			if !removed {
				return a.printMessage("Not logged in: %s doesn't exist.", path)
			}
			return a.printMessage("Removed %s.", path)
		},
	}
}

// readSecret reads a line from stdin, without echoing it if stdin is a
// terminal.
func (a *app) readSecret(prompt string) (string, error) {
	if f, ok := a.stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		_, _ = fmt.Fprint(a.stderr, prompt)
		secret, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(a.stderr)
		return string(secret), err
	}
	return readLine(a.stdin)
}

// readLine reads the first line from `r`, without the line ending.
func readLine(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

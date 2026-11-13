package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newBackupCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "backup <file>",
		Short: "Download a backup of the server's database",
		Long: `Download a backup of the server's database to <file>, or to stdout if it's "-".

This needs an API key with the admin scope. The backup contains every secret
on the server, such as TLS and SSH keys and user credentials, so keep it safe.
The file is created with mode 0600, and it's only replaced once the download
has completed.`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.newClient()
			if err != nil {
				return err
			}
			file := args[0]
			if file == "-" {
				_, err := c.downloadBackup(cmd.Context(), a.stdout)
				return err
			}

			// Download to a temporary file next to the target, so that a
			// failed download doesn't replace an earlier backup.
			tmp, err := os.CreateTemp(filepath.Dir(file), "."+filepath.Base(file)+".*.tmp")
			if err != nil {
				return err
			}
			defer func() { _ = os.Remove(tmp.Name()) }()
			size, err := c.downloadBackup(cmd.Context(), tmp)
			if err == nil {
				err = tmp.Sync()
			}
			if cerr := tmp.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
			if err := os.Rename(tmp.Name(), file); err != nil {
				return err
			}

			if a.output == outputJSON {
				return a.printJSON(map[string]any{"file": file, "size": size})
			}
			return a.printMessage("Wrote a %s backup to %s.", formatSize(size), file)
		},
	}
}

func formatSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

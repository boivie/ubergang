package main

import (
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBackup(t *testing.T) {
	srv := newTestServer(t)
	admin, _ := srv.fixture.CreateAdmin("backup-admin@example.com")
	key, rr := srv.fixture.CreateApiKey(admin, "backup", "admin")
	if rr.Code != http.StatusOK {
		t.Fatalf("Creating admin key: %d %s", rr.Code, rr.Body.String())
	}
	useServer(t, srv, key.Token)

	file := filepath.Join(t.TempDir(), "ubergang.db")
	out := mustRun(t, "", "backup", file, "-o", "json")
	var res struct {
		File string `json:"file"`
		Size int64  `json:"size"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("Invalid JSON %q: %v", out, err)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if res.File != file || res.Size != info.Size() {
		t.Errorf("Unexpected output %+v for a %d byte file", res, info.Size())
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Errorf("Expected mode 0600, got %v", info.Mode().Perm())
	}
	restored, err := db.New(log.NewLogger(log.Fields{}), file)
	if err != nil {
		t.Fatalf("Opening the backup: %v", err)
	}
	defer restored.Close()
	if _, err := restored.GetUserByEmail("backup-admin@example.com"); err != nil {
		t.Errorf("Expected the user in the backup: %v", err)
	}

	stdout := mustRun(t, "", "backup", "-")
	if int64(len(stdout)) != info.Size() {
		t.Errorf("Expected %d bytes on stdout, got %d", info.Size(), len(stdout))
	}
}

func TestBackupRequiresAdminScope(t *testing.T) {
	srv := newTestServer(t)
	useServer(t, srv, srv.writeKey)

	dir := t.TempDir()
	file := filepath.Join(dir, "ubergang.db")
	if err := os.WriteFile(file, []byte("earlier backup"), 0600); err != nil {
		t.Fatal(err)
	}
	if res := ugctl(t, "", "backup", file); res.code != exitUnauthorized {
		t.Errorf("Expected exit %d, got %d: %s", exitUnauthorized, res.code, res.stderr)
	}
	if data, _ := os.ReadFile(file); string(data) != "earlier backup" {
		t.Errorf("The earlier backup was changed: %q", data)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("Expected no temporary files to remain, got %v", entries)
	}
}

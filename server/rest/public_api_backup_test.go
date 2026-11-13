package rest

import (
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
)

func TestPublicBackupRequiresAdminScope(t *testing.T) {
	f := setupPublicApiTest(t)

	rr := f.publicRequest("GET", "/api/v1/backup", "", nil, nil)
	expectPublicError(t, rr, http.StatusUnauthorized, "unauthorized")
	for _, token := range []string{f.readToken, f.writeToken} {
		rr := f.publicRequest("GET", "/api/v1/backup", token, nil, nil)
		expectPublicError(t, rr, http.StatusForbidden, "insufficient_scope")
	}
}

func TestPublicBackup(t *testing.T) {
	f := setupPublicApiTest(t)
	created, rr := f.CreateApiKey(f.admin, "backup", "admin")
	if rr.Code != http.StatusOK || created.ApiKey.Scope != "admin" {
		t.Fatalf("Creating admin key: %d %s", rr.Code, rr.Body.String())
	}

	rr = f.publicRequest("GET", "/api/v1/backup", created.Token, nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("Unexpected Content-Type %q", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, `attachment; filename="ubergang-`) {
		t.Errorf("Unexpected Content-Disposition %q", cd)
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Unexpected Cache-Control %q", cc)
	}
	if cl := rr.Header().Get("Content-Length"); cl != strconv.Itoa(rr.Body.Len()) {
		t.Errorf("Content-Length %s doesn't match the body's %d bytes", cl, rr.Body.Len())
	}

	// The backup is a working database with the same data.
	file := path.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(file, rr.Body.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	restored, err := db.New(log.NewLogger(log.Fields{}), file)
	if err != nil {
		t.Fatalf("Opening the backup: %v", err)
	}
	defer restored.Close()
	if len(restored.ListApiKeys()) != len(f.Db.ListApiKeys()) {
		t.Errorf("Expected %d API keys in the backup, got %d",
			len(f.Db.ListApiKeys()), len(restored.ListApiKeys()))
	}
	if _, err := restored.GetUserByEmail("admin@example.com"); err != nil {
		t.Errorf("Expected the admin user in the backup: %v", err)
	}
}

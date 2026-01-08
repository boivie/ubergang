package wa

import (
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"encoding/hex"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAaGuid(t *testing.T) {
	config := &models.Configuration{
		AdminFqdn: "test.example.com",
	}

	log := log.NewLogger(log.Fields{})
	db, err := db.New(log, path.Join(t.TempDir(), "test.db"))
	if err != nil {
		panic(err)
	}

	t.Run("returns name for known aaguid", func(t *testing.T) {
		wa := New(config, db)
		aaguidString := "adce0002-35bc-c60a-648b-0b25f1f05503"
		bytes, err := hex.DecodeString(strings.ReplaceAll(aaguidString, "-", ""))
		require.NoError(t, err)

		name := wa.ResolveNameFromAaGuid(bytes)
		assert.Equal(t, "Chrome on Mac", name)
	})

	t.Run("returns name for unknown aaguid", func(t *testing.T) {
		wa := New(config, db)
		aaguidString := "11111111-1111-1111-1111-111111111111"
		bytes, err := hex.DecodeString(strings.ReplaceAll(aaguidString, "-", ""))
		require.NoError(t, err)

		name := wa.ResolveNameFromAaGuid(bytes)
		assert.Equal(t, "Unnamed passkey", name)
	})

	t.Run("returns name for invalid aaguid", func(t *testing.T) {
		wa := New(config, db)

		name := wa.ResolveNameFromAaGuid(make([]byte, 4))
		assert.Equal(t, "Unnamed passkey", name)
	})
}

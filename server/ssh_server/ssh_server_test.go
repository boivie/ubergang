package ssh_server

import (
	"boivie/ubergang/server/auth"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"path"
	"testing"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAuthenticate(t *testing.T) {
	now := time.Now()
	for name, tc := range map[string]struct {
		expiresAt *timestamppb.Timestamp
		valid     bool
	}{
		"unconfirmed": {nil, false},
		"expired":     {timestamppb.New(now.Add(-time.Second)), false},
		"valid":       {timestamppb.New(now.Add(time.Hour)), true},
	} {
		t.Run(name, func(t *testing.T) {
			logger := log.NewLogger(log.Fields{})
			database, err := db.New(logger, path.Join(t.TempDir(), "test.db"))
			require.NoError(t, err)
			user, _, err := auth.MakeUser(database, "user@example.com", "User", false, nil)
			require.NoError(t, err)

			pub, _, err := ed25519.GenerateKey(rand.Reader)
			require.NoError(t, err)
			pubKey, err := gossh.NewPublicKey(pub)
			require.NoError(t, err)
			fingerprint := sha256.Sum256(pubKey.Marshal())
			_, err = database.UpdateSshKey("key-id", func(_ *models.SshKey) (*models.SshKey, error) {
				return &models.SshKey{
					Id:                "key-id",
					UserId:            user.Id,
					Sha256Fingerprint: fingerprint[:],
					ExpiresAt:         tc.expiresAt,
				}, nil
			})
			require.NoError(t, err)

			s := New(logger, &models.Configuration{}, database, nil)
			key, keyUser, err := s.authenticate(ssh.PublicKey(pubKey), now)
			if tc.valid {
				require.NoError(t, err)
				assert.Equal(t, "key-id", key.Id)
				assert.Equal(t, user.Id, keyUser.Id)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestAuthenticateUnknownKey(t *testing.T) {
	logger := log.NewLogger(log.Fields{})
	database, err := db.New(logger, path.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	pubKey, err := gossh.NewPublicKey(pub)
	require.NoError(t, err)

	s := New(logger, &models.Configuration{}, database, nil)
	_, _, err = s.authenticate(pubKey, time.Now())
	assert.Error(t, err)
}

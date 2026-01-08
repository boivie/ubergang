package rest

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestProposeSshKey(t *testing.T) {
	f := CreateFixture(t)
	cookie, _ := f.CreateUser("test")

	// Enroll a credential
	enrollResp, err := f.StartEnroll(cookie)
	require.NoError(t, err)
	enrollReq := enrollResp.EnrollRequest
	_, res := f.GenerateCredential(enrollReq)
	_, err = f.FinishEnroll(cookie, enrollReq.Token, res)
	require.NoError(t, err, "FinishEnroll failed")

	key := f.createSshKey(cookie, "SSH-key-1")
	pubKey, _, _ := ed25519.GenerateKey(rand.Reader)
	sshPubKey, _ := ssh.NewPublicKey(pubKey)

	b := &bytes.Buffer{}
	b.WriteString(sshPubKey.Type())
	b.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, b)
	_, _ = e.Write(sshPubKey.Marshal())
	_ = e.Close()
	b.WriteByte(' ')
	b.WriteString("SSH-key-1")
	b.WriteByte('\n')
	fingerprint := sha256.Sum256(sshPubKey.Marshal())
	fingerprintBase64 := base64.RawURLEncoding.EncodeToString(fingerprint[:])

	confirmURL, err := f.proposeSshKey(key.KeyID, "SECRET", b.String())
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("https://test.example.com/ssh/%s", key.KeyID), confirmURL)

	user := f.getUser(cookie, "me")
	require.Len(t, user.SSHKeys, 1, "Expected SSH keys to be 1")
	assert.Equal(t, key.KeyID, user.SSHKeys[0].ID)
	assert.Equal(t, fingerprintBase64, user.SSHKeys[0].Sha256Fingerprint)
}

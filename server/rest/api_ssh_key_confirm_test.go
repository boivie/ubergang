package rest

import (
	"boivie/ubergang/server/auth"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestConfirmSshKey(t *testing.T) {
	f := CreateFixture(t)
	cookie, _ := f.CreateUser("test")
	resp, err := f.StartEnroll(cookie)
	require.NoError(t, err)
	request := resp.EnrollRequest
	cred, res := f.GenerateCredential(request)
	_, err = f.FinishEnroll(cookie, request.Token, res)
	require.NoError(t, err)

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

	_, _ = f.proposeSshKey(key.KeyID, "SECRET", b.String())

	confirmResp := f.requestConfirmSshKey(cookie, key.KeyID)
	if confirmResp.Authenticate == nil {
		t.Fatal("Expected Authenticate to be set")
	}

	req := confirmResp.Authenticate
	assertionResponse := f.SignAssertionRequest(&req.AssertionRequest, request.Options.User.ID, &cred)
	finalConfirmResp := f.confirmSshKey(cookie, key.KeyID, req.Token, assertionResponse)
	if finalConfirmResp.Result == nil {
		t.Error("Expected result to be set")
	}
	expiresAt, err := time.Parse(time.RFC3339, finalConfirmResp.Result.ExpiresAt)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(auth.SshKeyLifetime), expiresAt, time.Minute)

	// Proposing a new public key for it requires confirming it again.
	_, _ = f.proposeSshKey(key.KeyID, "SECRET", b.String())
	stored, err := f.Db.GetSshKey(key.KeyID)
	require.NoError(t, err)
	assert.Nil(t, stored.ConfirmedAt)
	assert.Nil(t, stored.ExpiresAt)
}

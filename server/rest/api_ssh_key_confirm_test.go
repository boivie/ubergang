package rest

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

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
	f.FinishEnroll(cookie, request.Token, res)

	key := f.createSshKey(cookie, "SSH-key-1")
	pubKey, _, _ := ed25519.GenerateKey(rand.Reader)
	sshPubKey, _ := ssh.NewPublicKey(pubKey)

	b := &bytes.Buffer{}
	b.WriteString(sshPubKey.Type())
	b.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, b)
	e.Write(sshPubKey.Marshal())
	e.Close()
	b.WriteByte(' ')
	b.WriteString("SSH-key-1")
	b.WriteByte('\n')

	f.proposeSshKey(key.KeyID, "SECRET", b.String())

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
	if finalConfirmResp.Result.ExpiresAt == "" {
		t.Errorf("Expected expiresAt to be set")
	}
}

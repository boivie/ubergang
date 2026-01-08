package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSshKey(t *testing.T) {
	f := CreateFixture(t)
	cookie, _ := f.CreateUser("test")

	// Enroll a credential
	enrollResp, err := f.StartEnroll(cookie)
	require.NoError(t, err)
	enrollReq := enrollResp.EnrollRequest
	_, res := f.GenerateCredential(enrollReq)
	_, err = f.FinishEnroll(cookie, enrollReq.Token, res)
	require.NoError(t, err, "FinishEnroll failed")

	createKeyResp := f.createSshKey(cookie, "SSH-key-1")
	assert.NotEmpty(t, createKeyResp.KeyID, "Expected keyID to be set")

	user := f.getUser(cookie, "me")
	require.Len(t, user.SSHKeys, 1, "Expected SSH keys to be 1")
	assert.Equal(t, createKeyResp.KeyID, user.SSHKeys[0].ID)
	assert.Empty(t, user.SSHKeys[0].Sha256Fingerprint, "Expected fingerprint to be empty")
}

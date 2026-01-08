package rest

import (
	"boivie/ubergang/server/common"
	"boivie/ubergang/server/models"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCredential(t *testing.T) {
	t.Run("deletes existing credential", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")
		user := f.getUser(cookie, "me")

		// Create a credential directly in the DB for this test.
		// In a real scenario, this would be created through the enrollment flow.
		cred := &models.Credential{
			Id:     "cred-" + common.MakeRandomID(),
			UserId: user.ID,
		}
		err := f.Db.UpdateCredential(cred.Id, func(old *models.Credential) (*models.Credential, error) {
			return cred, nil
		})
		require.NoError(t, err, "Failed to create test credential")

		// Delete the credential
		rr := f.request("DELETE", "/api/credential/"+cred.Id, nil, cookie, nil)
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify it's gone
		_, err = f.Db.GetCredential(cred.Id)
		assert.Error(t, err, "Expected credential to be deleted, but it was found")
	})

	t.Run("returns not found for non-existent credential", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Delete a non-existent credential
		rr := f.request("DELETE", "/api/credential/non-existent-id", nil, cookie, nil)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("requires authentication", func(t *testing.T) {
		f := CreateFixture(t)
		// No user, no cookie

		// Delete a credential without authentication
		rr := f.request("DELETE", "/api/credential/any-id", nil, nil, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartEnroll(t *testing.T) {
	t.Run("successfully starts enrollment for a logged in user", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		resp, err := f.StartEnroll(cookie)

		require.NoError(t, err)
		require.NotNil(t, resp.EnrollRequest)
		assert.NotEmpty(t, resp.EnrollRequest.Token)
		assert.NotEmpty(t, resp.EnrollRequest.Options.Challenge)
		assert.Equal(t, "test.example.com", resp.EnrollRequest.Options.RP.ID)
		assert.Equal(t, "test@example.com", resp.EnrollRequest.Options.User.Name)
	})

	t.Run("fails if user is not logged in", func(t *testing.T) {
		f := CreateFixture(t)

		// Make request without a session cookie
		resp, err := f.StartEnroll(nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 403")
		assert.Nil(t, resp.EnrollRequest)
		assert.Nil(t, resp.Error)
	})
}

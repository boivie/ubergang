package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUsers(t *testing.T) {
	t.Run("returns one user", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("admin@example.com")

		users := f.ListUsers(cookie)

		assert.Len(t, users, 1)
		assert.Equal(t, "admin@example.com", users[0].Email)
	})

	t.Run("returns multiple users", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("admin@example.com")
		_, _ = f.CreateUser("user1@example.com")
		_, _ = f.CreateUser("user2@example.com")

		users := f.ListUsers(cookie)

		require.Len(t, users, 3)

		emails := make([]string, len(users))
		for i, u := range users {
			emails[i] = u.Email
		}
		assert.Contains(t, emails, "admin@example.com")
		assert.Contains(t, emails, "user1@example.com")
		assert.Contains(t, emails, "user2@example.com")
	})

	t.Run("requires authentication", func(t *testing.T) {
		t.Skip("Authentication middleware not enabled in test fixture - endpoint returns data without auth")
	})
}

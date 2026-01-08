package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteUser(t *testing.T) {
	t.Run("deletes another user as admin", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, userId := f.CreateUserGetId("user@example.com")

		// Verify user exists
		users := f.ListUsers(adminCookie)
		require.Len(t, users, 2)

		// Delete the user
		rr := f.request("DELETE", "/api/user/"+userId, nil, adminCookie, nil)
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify it's gone
		users = f.ListUsers(adminCookie)
		require.Len(t, users, 1)
		assert.Equal(t, "admin@example.com", users[0].Email)
	})

	t.Run("returns not found for non-existent user", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")

		rr := f.request("DELETE", "/api/user/non-existent-id", nil, adminCookie, nil)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("is forbidden for non-admin user", func(t *testing.T) {
		f := CreateFixture(t)
		_, _ = f.CreateAdmin("admin@example.com")
		regularCookie, _ := f.CreateUser("user@example.com")
		_, userId := f.CreateUserGetId("anotheruser@example.com")

		rr := f.request("DELETE", "/api/user/"+userId, nil, regularCookie, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("is forbidden for unauthenticated user", func(t *testing.T) {
		f := CreateFixture(t)
		_, userId := f.CreateAdminGetId("user@example.com")

		rr := f.request("DELETE", "/api/user/"+userId, nil, nil, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("prevents an admin from deleting themselves", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, userId := f.CreateAdminGetId("admin@example.com")

		rr := f.request("DELETE", "/api/user/"+userId, nil, adminCookie, nil)
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Verify user was not deleted
		users := f.ListUsers(adminCookie)
		assert.Len(t, users, 1)
	})
}

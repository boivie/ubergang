package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	t.Run("creates a user as admin", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")

		req := api.ApiCreateUserRequest{
			Email: "newuser@example.com",
		}

		var resp api.ApiCreateUserResponse
		rr := f.request("POST", "/api/user", req, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NotEmpty(t, resp.ID)

		// Verify user was created
		users := f.ListUsers(adminCookie)
		require.Len(t, users, 2)

		var newUser *api.ApiUser
		for _, u := range users {
			if u.Email == "newuser@example.com" {
				newUser = &u
				break
			}
		}
		require.NotNil(t, newUser)
		assert.Equal(t, "newuser@example.com", newUser.Email)
		assert.Equal(t, resp.ID, newUser.ID)
		assert.False(t, newUser.IsAdmin)
	})

	t.Run("returns bad request for empty email", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")

		req := api.ApiCreateUserRequest{
			Email: "",
		}

		rr := f.request("POST", "/api/user", req, adminCookie, nil)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("is forbidden for non-admin user", func(t *testing.T) {
		f := CreateFixture(t)
		_, _ = f.CreateAdmin("admin@example.com")
		regularCookie, _ := f.CreateUser("user@example.com")

		req := api.ApiCreateUserRequest{
			Email: "newuser@example.com",
		}

		rr := f.request("POST", "/api/user", req, regularCookie, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("is forbidden for unauthenticated user", func(t *testing.T) {
		f := CreateFixture(t)

		req := api.ApiCreateUserRequest{
			Email: "newuser@example.com",
		}

		rr := f.request("POST", "/api/user", req, nil, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

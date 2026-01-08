package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateUser(t *testing.T) {
	t.Run("updates user email as admin", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, userId := f.CreateUserGetId("user@example.com")

		newEmail := "updated@example.com"
		req := api.ApiUpdateUserRequest{
			Email: &newEmail,
		}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+userId, req, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify user was updated
		users := f.ListUsers(adminCookie)
		var updatedUser *api.ApiUser
		for _, u := range users {
			if u.ID == userId {
				updatedUser = &u
				break
			}
		}
		require.NotNil(t, updatedUser)
		assert.Equal(t, "updated@example.com", updatedUser.Email)
		assert.False(t, updatedUser.IsAdmin)
	})

	t.Run("updates user display name as admin", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, userId := f.CreateUserGetId("user@example.com")

		newDisplayName := "Updated User Name"
		req := api.ApiUpdateUserRequest{
			DisplayName: &newDisplayName,
		}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+userId, req, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify user was updated
		users := f.ListUsers(adminCookie)
		var updatedUser *api.ApiUser
		for _, u := range users {
			if u.ID == userId {
				updatedUser = &u
				break
			}
		}
		require.NotNil(t, updatedUser)
		assert.Equal(t, "Updated User Name", updatedUser.DisplayName)
		assert.Equal(t, "user@example.com", updatedUser.Email)
	})

	t.Run("promotes user to admin", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, userId := f.CreateUserGetId("user@example.com")

		admin := true
		req := api.ApiUpdateUserRequest{
			Admin: &admin,
		}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+userId, req, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify user was promoted to admin
		users := f.ListUsers(adminCookie)
		var updatedUser *api.ApiUser
		for _, u := range users {
			if u.ID == userId {
				updatedUser = &u
				break
			}
		}
		require.NotNil(t, updatedUser)
		assert.True(t, updatedUser.IsAdmin)
	})

	t.Run("demotes admin to regular user", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, userId := f.CreateAdminGetId("useradmin@example.com")

		admin := false
		req := api.ApiUpdateUserRequest{
			Admin: &admin,
		}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+userId, req, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify user was demoted
		users := f.ListUsers(adminCookie)
		var updatedUser *api.ApiUser
		for _, u := range users {
			if u.ID == userId {
				updatedUser = &u
				break
			}
		}
		require.NotNil(t, updatedUser)
		assert.False(t, updatedUser.IsAdmin)
	})

	t.Run("updates multiple fields at once", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, userId := f.CreateUserGetId("user@example.com")

		newEmail := "newuser@example.com"
		newDisplayName := "New User Name"
		admin := true
		req := api.ApiUpdateUserRequest{
			Email:       &newEmail,
			DisplayName: &newDisplayName,
			Admin:       &admin,
		}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+userId, req, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify all fields were updated
		users := f.ListUsers(adminCookie)
		var updatedUser *api.ApiUser
		for _, u := range users {
			if u.ID == userId {
				updatedUser = &u
				break
			}
		}
		require.NotNil(t, updatedUser)
		assert.Equal(t, "newuser@example.com", updatedUser.Email)
		assert.Equal(t, "New User Name", updatedUser.DisplayName)
		assert.True(t, updatedUser.IsAdmin)
	})

	t.Run("returns not found for non-existent user", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")

		newEmail := "updated@example.com"
		req := api.ApiUpdateUserRequest{
			Email: &newEmail,
		}

		rr := f.request("POST", "/api/user/non-existent-id", req, adminCookie, nil)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("is forbidden for non-admin user", func(t *testing.T) {
		f := CreateFixture(t)
		_, _ = f.CreateAdmin("admin@example.com")
		regularCookie, _ := f.CreateUser("user@example.com")
		_, userId := f.CreateUserGetId("anotheruser@example.com")

		newEmail := "updated@example.com"
		req := api.ApiUpdateUserRequest{
			Email: &newEmail,
		}

		rr := f.request("POST", "/api/user/"+userId, req, regularCookie, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("is forbidden for unauthenticated user", func(t *testing.T) {
		f := CreateFixture(t)
		_, userId := f.CreateAdminGetId("user@example.com")

		newEmail := "updated@example.com"
		req := api.ApiUpdateUserRequest{
			Email: &newEmail,
		}

		rr := f.request("POST", "/api/user/"+userId, req, nil, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("handles empty request body", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, userId := f.CreateUserGetId("user@example.com")

		req := api.ApiUpdateUserRequest{}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+userId, req, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify user was not changed
		users := f.ListUsers(adminCookie)
		var user *api.ApiUser
		for _, u := range users {
			if u.ID == userId {
				user = &u
				break
			}
		}
		require.NotNil(t, user)
		assert.Equal(t, "user@example.com", user.Email)
		assert.False(t, user.IsAdmin)
	})

	// Self-update tests
	t.Run("allows regular user to update their own email", func(t *testing.T) {
		f := CreateFixture(t)
		_, _ = f.CreateAdmin("admin@example.com")
		userCookie, userId := f.CreateUserGetId("user@example.com")

		newEmail := "newemail@example.com"
		req := api.ApiUpdateUserRequest{
			Email: &newEmail,
		}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+userId, req, userCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify user's email was updated
		user := f.getUser(userCookie, userId)
		assert.Equal(t, "newemail@example.com", user.Email)
		assert.False(t, user.IsAdmin)
	})

	t.Run("allows regular user to update their own display name", func(t *testing.T) {
		f := CreateFixture(t)
		_, _ = f.CreateAdmin("admin@example.com")
		userCookie, userId := f.CreateUserGetId("user@example.com")

		newDisplayName := "New Display Name"
		req := api.ApiUpdateUserRequest{
			DisplayName: &newDisplayName,
		}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+userId, req, userCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify user's display name was updated
		user := f.getUser(userCookie, userId)
		assert.Equal(t, "New Display Name", user.DisplayName)
		assert.Equal(t, "user@example.com", user.Email)
		assert.False(t, user.IsAdmin)
	})

	t.Run("allows regular user to update both their email and display name", func(t *testing.T) {
		f := CreateFixture(t)
		_, _ = f.CreateAdmin("admin@example.com")
		userCookie, userId := f.CreateUserGetId("user@example.com")

		newEmail := "updated@example.com"
		newDisplayName := "Updated User"
		req := api.ApiUpdateUserRequest{
			Email:       &newEmail,
			DisplayName: &newDisplayName,
		}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+userId, req, userCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify both fields were updated
		user := f.getUser(userCookie, userId)
		assert.Equal(t, "updated@example.com", user.Email)
		assert.Equal(t, "Updated User", user.DisplayName)
		assert.False(t, user.IsAdmin)
	})

	t.Run("prevents regular user from changing their admin status", func(t *testing.T) {
		f := CreateFixture(t)
		_, _ = f.CreateAdmin("admin@example.com")
		userCookie, userId := f.CreateUserGetId("user@example.com")

		admin := true
		req := api.ApiUpdateUserRequest{
			Admin: &admin,
		}

		rr := f.request("POST", "/api/user/"+userId, req, userCookie, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)

		// Verify user is still not admin
		user := f.getUser(userCookie, userId)
		assert.False(t, user.IsAdmin)
	})

	t.Run("prevents regular user from attempting to demote themselves", func(t *testing.T) {
		f := CreateFixture(t)
		_, _ = f.CreateAdmin("admin@example.com")
		userCookie, userId := f.CreateUserGetId("user@example.com")

		admin := false
		req := api.ApiUpdateUserRequest{
			Admin: &admin,
		}

		rr := f.request("POST", "/api/user/"+userId, req, userCookie, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("allows admin to update their own profile including admin status", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, adminId := f.CreateAdminGetId("admin@example.com")

		newEmail := "newadmin@example.com"
		newDisplayName := "New Admin Name"
		admin := true
		req := api.ApiUpdateUserRequest{
			Email:       &newEmail,
			DisplayName: &newDisplayName,
			Admin:       &admin,
		}

		var resp api.ApiUpdateUserResponse
		rr := f.request("POST", "/api/user/"+adminId, req, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify admin updated themselves
		user := f.getUser(adminCookie, adminId)
		assert.Equal(t, "newadmin@example.com", user.Email)
		assert.Equal(t, "New Admin Name", user.DisplayName)
		assert.True(t, user.IsAdmin)
	})
}

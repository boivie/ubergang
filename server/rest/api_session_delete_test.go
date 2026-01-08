package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteSession(t *testing.T) {
	t.Run("deletes own session", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")
		user := f.getUser(cookie, "me")

		// Get session to delete
		sessions := f.Db.ListSessions(user.ID)
		require.Len(t, sessions, 1)
		sessionToDelete := sessions[0]

		// Delete the session
		rr := f.request("DELETE", "/api/session/"+sessionToDelete.Id, nil, cookie, nil)
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify it's gone
		_, _, err := f.Db.GetSession(sessionToDelete.Id)
		assert.Error(t, err) // Should be a "not found" error
	})

	t.Run("cannot delete another user's session", func(t *testing.T) {
		f := CreateFixture(t)
		cookieUserA, _ := f.CreateUser("user.a@example.com")

		// Create user B and get their session
		cookieUserB, _ := f.CreateUser("user.b@example.com")
		userB := f.getUser(cookieUserB, "me")
		sessions := f.Db.ListSessions(userB.ID)
		require.Len(t, sessions, 1)
		sessionOfUserB := sessions[0]

		// User A tries to delete User B's session
		rr := f.request("DELETE", "/api/session/"+sessionOfUserB.Id, nil, cookieUserA, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)

		// Verify session of user B still exists
		_, _, err := f.Db.GetSession(sessionOfUserB.Id)
		assert.NoError(t, err)
	})

	t.Run("returns not found for non-existent session", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Delete a non-existent session
		rr := f.request("DELETE", "/api/session/non-existent-id", nil, cookie, nil)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("requires authentication", func(t *testing.T) {
		f := CreateFixture(t)
		// No user, no cookie

		// Delete a session without authentication
		rr := f.request("DELETE", "/api/session/any-id", nil, nil, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

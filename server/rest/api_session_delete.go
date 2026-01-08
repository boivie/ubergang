package rest

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	sessionId := mux.Vars(r)["id"]

	// Users should only be able to delete their own sessions.
	// Let's get the session to be deleted and check the user ID.
	_, sessionToDelete, err := s.db.GetSession(sessionId)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !user.IsAdmin && sessionToDelete.UserId != user.Id {
		// This is a security risk. A user is trying to delete another user's session.
		s.log.Warnf("User %s trying to delete session %s belonging to user %s", user.Id, sessionId, sessionToDelete.UserId)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = s.db.DeleteSession(sessionId)

	if err != nil {
		// This would be a server error
		s.log.Errorf("Error deleting session %s: %v", sessionId, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

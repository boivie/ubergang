package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleUserUpdate(w http.ResponseWriter, r *http.Request) {
	sessionUser, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	userId := mux.Vars(r)["id"]
	isUpdatingSelf := sessionUser.Id == userId

	// Non-admins can only update themselves
	if !sessionUser.IsAdmin && !isUpdatingSelf {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	var req api.ApiUpdateUserRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	// Non-admins cannot change admin status
	if !sessionUser.IsAdmin && req.Admin != nil {
		http.Error(w, "Not authorized to change admin status", http.StatusForbidden)
		return
	}

	// Non-admins cannot change allowed hosts
	if !sessionUser.IsAdmin && req.AllowedHosts != nil {
		http.Error(w, "Not authorized to change allowed hosts", http.StatusForbidden)
		return
	}

	err = s.db.UpdateUser(userId, func(old *models.User) (*models.User, error) {
		if old == nil {
			return nil, fmt.Errorf("user not found")
		}

		if req.Email != nil {
			old.Email = *req.Email
		}
		if req.DisplayName != nil {
			old.DisplayName = *req.DisplayName
		}
		if req.Admin != nil {
			old.IsAdmin = *req.Admin
		}
		if req.AllowedHosts != nil {
			old.AllowedHosts = *req.AllowedHosts
		}
		return old, nil
	})

	if err != nil {
		if err.Error() == "user not found" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		s.log.Warnf("Failed to update user: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	jsonify(w, api.ApiUpdateUserResponse{})
}

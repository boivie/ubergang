package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
)

func (s *ApiModule) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	var req api.ApiCreateUserRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	newUser, _, err := s.auth.CreateUser(req.Email, req.Email, false, []string{})
	if err != nil {
		s.log.Warnf("Failed to create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusBadRequest)
		return
	}

	jsonify(w, api.ApiCreateUserResponse{
		ID: newUser.Id,
	})
}

package rest

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	userId := mux.Vars(r)["id"]

	if user.Id == userId {
		http.Error(w, "You cannot delete yourself", http.StatusBadRequest)
		return
	}

	err = s.db.DeleteUser(userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

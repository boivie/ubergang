package rest

import (
	"boivie/ubergang/server/models"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleCredentialDelete(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	id := mux.Vars(r)["id"]

	err = s.db.UpdateCredential(id, func(old *models.Credential) (*models.Credential, error) {
		if old == nil {
			return nil, errors.New("credential not found")
		}
		if !user.IsAdmin && old.UserId != user.Id {
			return nil, errors.New("not authorized")
		}
		return nil, nil
	})

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

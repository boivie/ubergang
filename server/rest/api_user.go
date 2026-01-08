package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleUserGet(w http.ResponseWriter, r *http.Request) {
	sessionUser, session, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	var user *models.User
	var currentSession *api.ApiSession = nil
	userId := mux.Vars(r)["id"]
	if userId == "me" {
		user = sessionUser
		as := ToApiSession(session)
		currentSession = &as
	} else {
		user, err = s.db.GetUserById(userId)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	au := api.ApiUser{
		ID:             user.Id,
		Email:          user.Email,
		DisplayName:    user.DisplayName,
		AllowedHosts:   user.AllowedHosts,
		IsAdmin:        user.IsAdmin,
		Credentials:    make([]api.ApiCredential, 0),
		Sessions:       make([]api.ApiSession, 0),
		SSHKeys:        make([]api.ApiSSHKey, 0),
		CurrentSession: currentSession,
	}

	for _, c := range s.db.ListCredentials(user.Id) {
		au.Credentials = append(au.Credentials, ToApiCredential(c))
	}
	for _, s := range s.db.ListSessions(user.Id) {
		au.Sessions = append(au.Sessions, ToApiSession(s))
	}
	for _, key := range s.db.ListSshKeys(user.Id) {
		au.SSHKeys = append(au.SSHKeys, ToApiSshKey(key))
	}
	jsonify(w, au)
}

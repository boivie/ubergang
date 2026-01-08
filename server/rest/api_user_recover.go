package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/common"
	"boivie/ubergang/server/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ApiModule) handleUserRecover(w http.ResponseWriter, r *http.Request) {
	sessionUser, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !sessionUser.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	userId := mux.Vars(r)["id"]
	if userId == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	pollId := common.MakeSigninRequestToken()

	err = s.db.UpdateUser(userId, func(user *models.User) (*models.User, error) {
		if user == nil {
			return nil, fmt.Errorf("user not found")
		}

		signinRequest := &models.SigninRequest{
			Id:        pollId,
			ExpiresAt: timestamppb.New(time.Now().Add(7 * 24 * time.Hour)),
			Confirmed: true,
		}

		user.SigninRequests = append(user.SigninRequests, signinRequest)

		return user, nil
	})

	if err != nil {
		s.log.Warnf("Failed to create recovery token for user %s: %v", userId, err)
		if err.Error() == "user not found" {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to create recovery token", http.StatusInternalServerError)
		}
		return
	}

	recoveryUrl := fmt.Sprintf("https://%s/signin/%s", s.config.AdminFqdn, pollId)
	s.log.Infof("Created recovery token for user %s", userId)

	jsonify(w, api.ApiUserRecoverResponse{
		RecoveryUrl: recoveryUrl,
	})
}

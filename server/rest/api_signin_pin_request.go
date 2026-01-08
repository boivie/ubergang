package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/common"
	"boivie/ubergang/server/models"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ApiModule) handleSigninPinRequest(w http.ResponseWriter, r *http.Request) {
	// Note: This endpoint should not be authenticated, as it's part of the
	// sign-in flow.

	var req api.ApiRequestSigninPinRequest
	err := parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	user, err := s.db.GetUserByEmail(req.Email)
	if err != nil {
		s.log.Warnf("User not found for %s", req.Email)
		jsonify(w, api.ApiRequestSigninPinResponse{
			Error: &api.ApiRequestSigninPinError{
				InvalidEmail: true}})
		return
	}

	pin, err := common.MakeSigninRequestPin()
	if err != nil {
		s.log.Errorf("Failed to generate PIN: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pollId := uuid.New().String()

	err = s.db.UpdateUser(user.Id, func(old *models.User) (*models.User, error) {
		if old == nil {
			return nil, errors.New("user not found")
		}
		// TODO: Rate limit
		old.SigninRequests = append(old.SigninRequests, &models.SigninRequest{
			Id:        pollId,
			Pin:       pin,
			ExpiresAt: timestamppb.New(time.Now().Add(30 * time.Minute)),
			UserAgent: req.UserAgent,
			Ip:        common.ReadUserIP(r),
		})
		return old, nil
	})
	if err != nil {
		s.log.Warnf("Failed to update user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonify(w, api.ApiRequestSigninPinResponse{
		ID: pollId,
	})
}

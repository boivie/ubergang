package auth

import (
	"boivie/ubergang/server/common"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"net"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Auth struct {
	log *log.Log
	db  *db.DB
}

func New(log *log.Log, db *db.DB) *Auth {
	return &Auth{log, db}
}

func (s *Auth) CreateUser(email string, displayName string, admin bool, allowedHosts []string) (user *models.User, pollId string, err error) {
	pollId = common.MakeSigninRequestToken()
	userId := common.MakeRandomID()

	user = &models.User{
		Id:           userId,
		Email:        email,
		DisplayName:  displayName,
		AllowedHosts: allowedHosts,
		IsAdmin:      admin,
		SigninRequests: []*models.SigninRequest{
			{
				Id:        pollId,
				ExpiresAt: timestamppb.New(time.Now().Add(7 * 24 * time.Hour)),
				Confirmed: true,
			},
		},
	}

	err = s.db.UpdateUser(user.Id, func(old *models.User) (*models.User, error) {
		if old != nil {
			return nil, errors.New("user already exists")
		}
		return user, nil
	})
	s.log.Infof("Created user %s with signin token %s", user.Email, pollId)

	return
}

func (s *Auth) CreateSession(userId, userAgent, remoteAddr string) (*models.Session, error) {
	user, err := s.db.GetUserById(userId)
	if err != nil {
		return nil, err
	}

	ip, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		remoteAddr = ip
	}

	now := time.Now()
	session := &models.Session{
		Id:         common.MakeRandomID(),
		UserId:     user.Id,
		Secret:     common.MakeRandomID(),
		UserAgent:  userAgent,
		RemoteAddr: remoteAddr,
		CreatedAt:  timestamppb.New(now),
	}

	err = s.db.UpdateSession(session.Id, func(old *models.Session) (*models.Session, error) {
		if old != nil {
			// Not supposed to happen!
			return nil, errors.New("session ID collision")
		}
		return session, nil
	})

	return session, err
}

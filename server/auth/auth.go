package auth

import (
	"boivie/ubergang/server/common"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"net"
	"slices"
	"strings"
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

func MakeUser(db *db.DB, email string, displayName string, admin bool, allowedHosts []string) (user *models.User, pollId string, err error) {
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

	err = db.UpdateUser(user.Id, func(old *models.User) (*models.User, error) {
		if old != nil {
			return nil, errors.New("user already exists")
		}
		return user, nil
	})

	return
}

func (s *Auth) CreateUser(email string, displayName string, admin bool, allowedHosts []string) (user *models.User, pollId string, err error) {
	user, pollId, err = MakeUser(s.db, email, displayName, admin, allowedHosts)
	if err != nil {
		return
	}
	s.log.Infof("Created user %s with signin token %s", user.Email, pollId)
	return
}

// CanAccessHost returns whether `user` may access the backend at `host`.
func CanAccessHost(user *models.User, host string) bool {
	return user.IsAdmin || slices.Contains(user.AllowedHosts, host)
}

// CreateSession creates a session for the user, valid on the admin host and
// on all backends the user can access.
func (s *Auth) CreateSession(userId, userAgent, remoteAddr string) (*models.Session, error) {
	return s.createSession(userId, userAgent, remoteAddr, func(*models.Session) {})
}

// CreateKioskSession creates a session for the user that is only valid on the
// backend at `host`, and only until `expiresAt` unless it's zero.
func (s *Auth) CreateKioskSession(userId, userAgent, remoteAddr, host, name string, expiresAt time.Time) (*models.Session, error) {
	return s.createSession(userId, userAgent, remoteAddr, func(session *models.Session) {
		session.RestrictedHost = strings.ToLower(host)
		session.Name = name
		if !expiresAt.IsZero() {
			session.ExpiresAt = timestamppb.New(expiresAt)
		}
	})
}

func (s *Auth) createSession(userId, userAgent, remoteAddr string, init func(*models.Session)) (*models.Session, error) {
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
	init(session)

	err = s.db.UpdateSession(session.Id, func(old *models.Session) (*models.Session, error) {
		if old != nil {
			// Not supposed to happen!
			return nil, errors.New("session ID collision")
		}
		return session, nil
	})

	return session, err
}

package auth

import (
	"boivie/ubergang/server/common"
	"boivie/ubergang/server/models"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/pem"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func GenerateSshKey() ([]byte, error) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	privKeyBytes, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return nil, err
	}
	privKeyPem := pem.EncodeToMemory(privKeyBytes)
	return privKeyPem, nil
}

func (s *Auth) CreateSshKey(userID string, name string) (*models.SshKey, error) {
	key := &models.SshKey{
		Id:        common.MakeRandomID(),
		UserId:    userID,
		Name:      name,
		CreatedAt: timestamppb.Now(),
	}

	return s.db.UpdateSshKey(key.Id, func(old *models.SshKey) (*models.SshKey, error) {
		if old != nil {
			return nil, errors.New("ID collision")
		}
		return key, nil
	})
}

func (s *Auth) UpdateSshKey(keyID, keySecret, pubKey string) (*models.SshKey, error) {
	parsedPubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubKey))
	if err != nil {
		return nil, err
	}
	fingerprint := sha256.Sum256(parsedPubKey.Marshal())
	s.log.Infof("Updated key %s with pub key fp %s", keyID, base64.RawURLEncoding.EncodeToString(fingerprint[:]))
	return s.db.UpdateSshKey(keyID, func(old *models.SshKey) (*models.SshKey, error) {
		if old == nil {
			return nil, errors.New("Unknown key")
		} else if old.HashedSecret == "" {
			// Never updated before.
			hashed, err := common.HashPassword(keySecret)
			if err != nil {
				return nil, err
			}
			old.HashedSecret = hashed
		} else {
			err := common.CheckPassword(old.HashedSecret, keySecret)
			if err != nil {
				return nil, err
			}
		}

		old.PublicKey = pubKey
		old.Sha256Fingerprint = fingerprint[:]
		old.ConfirmedAt = nil
		return old, nil
	})
}

func (s *Auth) ConfirmSshKeyUpdate(keyID string, now time.Time) (*models.SshKey, error) {
	return s.db.UpdateSshKey(keyID, func(old *models.SshKey) (*models.SshKey, error) {
		if old == nil {
			return nil, errors.New("Unknown key")
		}
		old.ConfirmedAt = timestamppb.New(now)
		return old, nil
	})
}

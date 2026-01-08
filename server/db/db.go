package db

import (
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

type DB struct {
	db  *bolt.DB
	log *log.Log
}

var BucketName = []byte("ug")

func New(log *log.Log, dbFile string) (*DB, error) {

	db, err := bolt.Open(dbFile, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(BucketName)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &DB{db, log}, nil
}

func (d *DB) Close() {
	d.db.Close()
}

func sshServertKeyKey() []byte {
	return []byte("ssh-server-key")
}

func selfSignedCertKey() []byte {
	return []byte("self-signed-cert")
}

func selfSignedKeyKey() []byte {
	return []byte("self-signed-key")
}

func configKey() []byte {
	return []byte("config")
}

func certKey(name string) []byte {
	return []byte(fmt.Sprintf("cert:%s", name))
}

func sessionKey(id string) []byte {
	return []byte(fmt.Sprintf("sess:%s", id))
}

func backendKey(fqdn string) []byte {
	return []byte(fmt.Sprintf("be:%s", fqdn))
}

func credentialKey(id string) []byte {
	return []byte(fmt.Sprintf("cred:%s", id))
}

func userKey(id string) []byte {
	return []byte(fmt.Sprintf("user:%s", id))
}

func sshKeyKey(id string) []byte {
	return []byte(fmt.Sprintf("ssh-key:%s", id))
}

func sshFingerprintKey(fingerprint []byte) []byte {
	b64 := base64.RawURLEncoding.EncodeToString(fingerprint)
	return []byte(fmt.Sprintf("ssh-fp:%s", b64))
}

func emailKey(email string) []byte {
	return []byte(fmt.Sprintf("email:%s", email))
}

func signinTokenKey(token string) []byte {
	return []byte(fmt.Sprintf("signin:%s", token))
}

func authenticationStateKey(stateId *uuid.UUID) []byte {
	return []byte(fmt.Sprintf("auth-state:%s", stateId.String()))
}

func mqttProfileKey(id string) []byte {
	return []byte(fmt.Sprintf("mqtt-profile:%s", id))
}

func mqttClientKey(id string) []byte {
	return []byte(fmt.Sprintf("mqtt-client:%s", id))
}

func (d *DB) GetCert(name string) (cert []byte, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get(certKey(name))
		if v == nil {
			return fmt.Errorf("failed to find key")
		}
		cert = append([]byte{}, v...)
		return nil
	})

	return
}

func (d *DB) GetSshServerKey() (key []byte, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get(sshServertKeyKey())
		if v == nil {
			return fmt.Errorf("failed to find key")
		}
		key = append([]byte{}, v...)
		return nil
	})

	return
}

func (d *DB) UpdateSshServerKey(data []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		return b.Put(sshServertKeyKey(), data)
	})
}

func (d *DB) GetSelfSignedCert() (cert []byte, key []byte, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		certData := b.Get(selfSignedCertKey())
		keyData := b.Get(selfSignedKeyKey())
		if certData == nil || keyData == nil {
			return fmt.Errorf("self-signed certificate not found")
		}
		cert = append([]byte{}, certData...)
		key = append([]byte{}, keyData...)
		return nil
	})
	return
}

func (d *DB) UpdateSelfSignedCert(cert []byte, key []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		if err := b.Put(selfSignedCertKey(), cert); err != nil {
			return err
		}
		return b.Put(selfSignedKeyKey(), key)
	})
}

func (d *DB) UpdateCert(name string, data []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		return b.Put(certKey(name), data)
	})
}

func (d *DB) DeleteCert(name string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		return b.Delete(certKey(name))
	})
}

func (d *DB) ListCertKeys() ([]string, error) {
	var keys []string
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		c := b.Cursor()

		prefix := []byte("cert:")
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			keyName := string(k[len(prefix):])
			keys = append(keys, keyName)
		}
		return nil
	})
	return keys, err
}

func (d *DB) DeleteCertsByPrefix(prefix string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		c := b.Cursor()

		certPrefix := []byte("cert:")
		var keysToDelete [][]byte

		for k, _ := c.Seek(certPrefix); k != nil && bytes.HasPrefix(k, certPrefix); k, _ = c.Next() {
			keyName := string(k[len(certPrefix):])
			if strings.HasPrefix(keyName, prefix) {
				keysToDelete = append(keysToDelete, append([]byte{}, k...))
			}
		}

		for _, key := range keysToDelete {
			if err := b.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}

func normalizeFqdn(fqdn string) string {
	return strings.ToLower(fqdn)
}

func (d *DB) GetBackend(fqdn string) (backend *models.Backend, err error) {
	fqdn = normalizeFqdn(fqdn)
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		backendBytes := b.Get(backendKey(fqdn))
		if backendBytes == nil {
			return fmt.Errorf("failed to find backend: %s", fqdn)
		}
		backend = &models.Backend{}
		return proto.Unmarshal(backendBytes, backend)
	})
	return
}

func (d *DB) GetSession(id string) (user *models.User, session *models.Session, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		sessionBytes := b.Get(sessionKey(id))
		if sessionBytes == nil {
			return fmt.Errorf("failed to find session: %s", id)
		}
		session = &models.Session{}
		err = proto.Unmarshal(sessionBytes, session)
		if err != nil {
			return err
		}

		userBytes := b.Get(userKey(session.UserId))
		if userBytes == nil {
			return fmt.Errorf("failed to find user: %s", session.UserId)
		}
		user = &models.User{}
		err = proto.Unmarshal(userBytes, user)
		if err != nil {
			return err
		}
		return nil
	})
	return
}

func (d *DB) ListSessions(userId string) (ret []*models.Session) {
	d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		c := b.Cursor()
		prefix := []byte(fmt.Sprintf("user-sess:%s:", userId))
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			sessionId := string(k[len(prefix):])
			v := b.Get(sessionKey(sessionId))
			sess := &models.Session{}
			err := proto.Unmarshal(v, sess)
			if err == nil {
				ret = append(ret, sess)
			}
		}
		return nil
	})
	return
}

func (d *DB) ListBackends() (ret []*models.Backend) {
	d.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(BucketName).Cursor()
		prefix := []byte("be:")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			cred := &models.Backend{}
			err := proto.Unmarshal(v, cred)
			if err == nil {
				ret = append(ret, cred)
			}
		}
		return nil
	})
	return
}

func (d *DB) ListUsers() (ret []*models.User) {
	d.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(BucketName).Cursor()
		prefix := []byte("user:")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			user := &models.User{}
			err := proto.Unmarshal(v, user)
			if err == nil {
				ret = append(ret, user)
			}
		}
		return nil
	})
	return
}

func (d *DB) DeleteBackend(fqdn string) error {
	fqdn = normalizeFqdn(fqdn)
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := backendKey(fqdn)
		return b.Delete(key)
	})
}

func (d *DB) DeleteUser(userId string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := userKey(userId)

		v := b.Get(key)
		if v == nil {
			return errors.New("user not found")
		}

		user := &models.User{}
		if err := proto.Unmarshal(v, user); err == nil {
			if user.Email != "" {
				b.Delete(emailKey(user.Email))
			}
		}
		// TODO: Delete associated objects like credentials, sessions etc

		return b.Delete(key)
	})
}

func (d *DB) DeleteSession(sessionId string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := sessionKey(sessionId)
		v := b.Get(key)
		if v == nil {
			return nil // Not found
		}

		// Get user ID from session
		session := &models.Session{}
		if err := proto.Unmarshal(v, session); err != nil {
			// Still try to delete the main key
		} else {
			// Delete user-session index key
			userSessionKey := []byte(fmt.Sprintf("user-sess:%s:%s", session.UserId, sessionId))
			err := b.Delete(userSessionKey)
			if err != nil {
				// Not fatal
			}
		}

		return b.Delete(key)
	})
}

func (d *DB) UpdateSession(sessionId string, update_fn func(old *models.Session) (*models.Session, error)) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := sessionKey(sessionId)
		v := b.Get(key)
		var old_obj *models.Session = nil
		if v != nil {
			old_obj = &models.Session{}
			err := proto.Unmarshal(v, old_obj)
			if err != nil {
				return err
			}
		}
		new_obj, err := update_fn(old_obj)
		if err != nil {
			return err
		}
		serialized, err := proto.Marshal(new_obj)
		if err != nil {
			return err
		}
		err = b.Put([]byte(fmt.Sprintf("user-sess:%s:%s", new_obj.UserId, new_obj.Id)), []byte{})
		if err != nil {
			return err
		}
		return b.Put(key, serialized)
	})
}

func (d *DB) GetConfiguration() (ret *models.Configuration, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get(configKey())
		if v == nil {
			return fmt.Errorf("failed to find configuration")
		}
		ret = &models.Configuration{}
		return proto.Unmarshal(v, ret)
	})
	return
}

func (d *DB) UpdateConfiguration(update_fn func(old *models.Configuration) (*models.Configuration, error)) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := configKey()
		v := b.Get(key)
		var old_obj *models.Configuration = nil
		if v != nil {
			old_obj = &models.Configuration{}
			err := proto.Unmarshal(v, old_obj)
			if err != nil {
				return err
			}
		}
		new_obj, err := update_fn(old_obj)
		if err != nil {
			return err
		}
		if new_obj == nil {
			panic("Can't delete configuration")
		}

		serialized, err := proto.Marshal(new_obj)
		if err != nil {
			return err
		}
		return b.Put(key, serialized)
	})
}

func (d *DB) GetUserByEmail(email string) (ret *models.User, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		userId := b.Get(emailKey(email))
		if userId == nil {
			return fmt.Errorf("failed to find email address")
		}
		userValue := b.Get(userKey(string(userId)))
		if userValue == nil {
			return fmt.Errorf("failed to find user")
		}
		ret = &models.User{}
		return proto.Unmarshal(userValue, ret)
	})
	return
}

func (d *DB) GetUserById(userId string) (ret *models.User, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get(userKey(userId))
		if v == nil {
			return fmt.Errorf("failed to find user")
		}
		ret = &models.User{}
		return proto.Unmarshal(v, ret)
	})
	return
}

// Diff method returns difference in keys between two sets
func diff(set1, set2 map[string]bool) map[string]bool {
	diff := make(map[string]bool)

	for k := range set1 {
		if !set2[k] {
			diff[k] = true
		}
	}

	return diff
}

func (d *DB) UpdateUser(userId string, update_fn func(old *models.User) (*models.User, error)) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := userKey(userId)
		v := b.Get(key)
		var old_obj *models.User = nil
		oldEmail := ""
		oldTokens := make(map[string]bool)
		newTokens := make(map[string]bool)
		if v != nil {
			old_obj = &models.User{}
			err := proto.Unmarshal(v, old_obj)
			if err != nil {
				return err
			}
			oldEmail = old_obj.Email
			for _, s := range old_obj.SigninRequests {
				oldTokens[s.Id] = true
			}
		}
		new_obj, err := update_fn(old_obj)
		if err != nil {
			return err
		}
		for _, s := range new_obj.SigninRequests {
			newTokens[s.Id] = true
		}
		// Validate that there isn't already a user with this e-mail address.
		if new_obj.Email != "" {
			existingUserId := b.Get(emailKey(new_obj.Email))
			if existingUserId != nil && string(existingUserId) != userId {
				return fmt.Errorf("e-mail address already mapped to another user: %s", string(existingUserId))
			}
		}
		if oldEmail != new_obj.Email {
			if oldEmail != "" {
				b.Delete(emailKey(oldEmail))
			}
			if new_obj.Email != "" {
				b.Put(emailKey(new_obj.Email), []byte(userId))
			}
		}
		for token := range diff(oldTokens, newTokens) {
			b.Delete(signinTokenKey(token))
		}
		for token := range diff(newTokens, oldTokens) {
			b.Put(signinTokenKey(token), []byte(userId))
		}
		serialized, err := proto.Marshal(new_obj)
		if err != nil {
			return err
		}
		return b.Put(key, serialized)
	})
}

func (d *DB) GetCredential(id string) (ret *models.Credential, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get([]byte(fmt.Sprintf("cred:%s", id)))
		if v == nil {
			return fmt.Errorf("failed to find credential")
		}
		ret = &models.Credential{}
		return proto.Unmarshal(v, ret)
	})
	return
}

func (d *DB) GetUserBySigninRequest(token string) (ret *models.User, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get(signinTokenKey(token))
		if v == nil {
			return fmt.Errorf("failed to find signin token")
		}
		v = b.Get(userKey(string(v)))
		if v == nil {
			return fmt.Errorf("failed to find user")
		}
		ret = &models.User{}
		return proto.Unmarshal(v, ret)
	})
	return
}

func (d *DB) ListCredentials(userId string) (ret []*models.Credential) {
	d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		c := b.Cursor()
		prefix := []byte(fmt.Sprintf("user-cred:%s:", userId))
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			credentialId := string(k[len(prefix):])
			v := b.Get(credentialKey(credentialId))
			if v != nil {
				cred := &models.Credential{}
				err := proto.Unmarshal(v, cred)
				if err == nil {
					ret = append(ret, cred)
				}
			}
		}
		return nil
	})
	return
}

func (d *DB) UpdateCredential(credentialId string, update_fn func(old *models.Credential) (*models.Credential, error)) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := credentialKey(credentialId)
		v := b.Get(key)
		var old_obj *models.Credential = nil
		if v != nil {
			old_obj = &models.Credential{}
			err := proto.Unmarshal(v, old_obj)
			if err != nil {
				return err
			}
		}
		new_obj, err := update_fn(old_obj)
		if err != nil {
			return err
		}
		if new_obj == nil {
			// Credential is to be deleted.
			if old_obj != nil {
				b.Delete([]byte(fmt.Sprintf("user-cred:%s:%s", old_obj.UserId, old_obj.Id)))
				b.Delete(key)
			}
			return nil
		}

		serialized, err := proto.Marshal(new_obj)
		if err != nil {
			return err
		}
		err = b.Put([]byte(fmt.Sprintf("user-cred:%s:%s", new_obj.UserId, new_obj.Id)), []byte{})
		if err != nil {
			return err
		}
		return b.Put(key, serialized)
	})
}

func (d *DB) UpdateBackend(fqdn string, update_fn func(old *models.Backend) (*models.Backend, error)) error {
	fqdn = normalizeFqdn(fqdn)
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := backendKey(fqdn)
		v := b.Get(key)
		var old_obj *models.Backend = nil
		if v != nil {
			old_obj = &models.Backend{}
			err := proto.Unmarshal(v, old_obj)
			if err != nil {
				return err
			}
		}
		new_obj, err := update_fn(old_obj)
		if err != nil {
			return err
		}
		if new_obj == nil {
			// Backend is to be deleted.
			if old_obj != nil {
				b.Delete(key)
			}
			return nil
		}
		if old_obj != nil && old_obj.Fqdn != new_obj.Fqdn {
			return fmt.Errorf("changing FQDN is currently not supported")
		}

		serialized, err := proto.Marshal(new_obj)
		if err != nil {
			return err
		}
		return b.Put(key, serialized)
	})
}

func (d *DB) ListSshKeys(userId string) (ret []*models.SshKey) {
	d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		c := b.Cursor()
		prefix := []byte(fmt.Sprintf("user-ssh-key:%s:", userId))
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			sshKeyId := string(k[len(prefix):])
			v := b.Get(sshKeyKey(sshKeyId))
			if v != nil {
				cred := &models.SshKey{}
				err := proto.Unmarshal(v, cred)
				if err == nil {
					ret = append(ret, cred)
				}
			}
		}
		return nil
	})
	return
}

func (d *DB) GetSshKey(id string) (ret *models.SshKey, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get(sshKeyKey(id))
		if v == nil {
			return fmt.Errorf("failed to find sshKey")
		}
		ret = &models.SshKey{}
		return proto.Unmarshal(v, ret)
	})
	return
}

func (d *DB) GetSshKeyByFingerprint(sha256Fingerprint []byte) (ret *models.SshKey, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		keyIdValue := b.Get(sshFingerprintKey(sha256Fingerprint))
		if keyIdValue == nil {
			return fmt.Errorf("failed to find fingerprint")
		}
		v := b.Get(sshKeyKey(string(keyIdValue)))
		if v == nil {
			return fmt.Errorf("failed to find ssh key")
		}
		ret = &models.SshKey{}
		return proto.Unmarshal(v, ret)
	})
	return
}

func (d *DB) UpdateSshKey(sshKeyId string, update_fn func(old *models.SshKey) (*models.SshKey, error)) (ret *models.SshKey, err error) {
	err = d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get(sshKeyKey(sshKeyId))
		var old_obj *models.SshKey = nil
		var oldFingerprint []byte
		if v != nil {
			old_obj = &models.SshKey{}
			err := proto.Unmarshal(v, old_obj)
			if err != nil {
				return err
			}
			oldFingerprint = old_obj.Sha256Fingerprint
		}
		ret, err = update_fn(old_obj)
		if err != nil {
			return err
		}
		serialized, err := proto.Marshal(ret)
		if err != nil {
			return err
		}
		if !bytes.Equal(oldFingerprint, ret.Sha256Fingerprint) {
			if oldFingerprint != nil {
				b.Delete(sshFingerprintKey(oldFingerprint))
			}
			b.Put(sshFingerprintKey(ret.Sha256Fingerprint), []byte(sshKeyId))
		}
		if old_obj == nil {
			b.Put([]byte(fmt.Sprintf("user-ssh-key:%s:%s", ret.UserId, ret.Id)), []byte{})
		}
		return b.Put(sshKeyKey(sshKeyId), serialized)
	})

	return
}

func (d *DB) StoreAuthenticationState(stateId *uuid.UUID, state *models.AuthenticationState) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := authenticationStateKey(stateId)
		v := b.Get(key)
		if v != nil {
			return errors.New("ID collision")
		}
		serialized, err := proto.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put(key, serialized)
	})
}

func (d *DB) ConsumeAuthenticationState(token string) (ret *models.AuthenticationState, err error) {
	stateUuid, err := uuid.Parse(token)
	if err != nil || stateUuid.Version() != 7 {
		return nil, errors.New("invalid token")
	}
	err = d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := authenticationStateKey(&stateUuid)
		v := b.Get(key)
		if v == nil {
			return errors.New("authentication state not found")
		}
		ret = &models.AuthenticationState{}
		err := proto.Unmarshal(v, ret)
		if err != nil {
			return err
		}
		return b.Delete(key)
	})

	return
}

func (d *DB) ClearDatabase() error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		c := b.Cursor()
		keysToDelete := [][]byte{}
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			key := string(k)
			if strings.HasPrefix(key, "cert:") {
				// Don't delete certs - they are expensive to regenerate.
			} else if key == "config" {
				// The configruation is required.
			} else if key == "ssh-server-key" {
				// Keep the SSH server key.
			} else {
				keysToDelete = append(keysToDelete, k)
			}
		}
		for _, k := range keysToDelete {
			err := b.Delete(k)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DB) BackupHttpHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := d.db.View(func(tx *bolt.Tx) error {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", `attachment; filename="ubergang.db"`)
			w.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
			_, err := tx.WriteTo(w)
			return err
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (d *DB) PerformPeriodicBackups(directory string, interval time.Duration) {
	for range time.Tick(interval) {
		var outputFile bytes.Buffer
		var originalSize int64
		filename := "ubergang-backup.db.gz"
		outGzip := gzip.NewWriter(&outputFile)
		err := d.db.View(func(tx *bolt.Tx) error {
			originalSize = tx.Size()
			_, err := tx.WriteTo(outGzip)
			return err
		})
		if err != nil {
			d.log.Warnf("Failed to backup database: %v", err)
			continue
		}
		err = outGzip.Close()
		if err != nil {
			d.log.Warnf("Failed to compress database: %v", err)
			continue
		}
		err = os.WriteFile(path.Join(directory, filename), outputFile.Bytes(), 0600)
		if err != nil {
			d.log.Warnf("Failed to write database backup: %v", err)
			continue
		}
		d.log.Infof("Database backed up as %s (%d -> %d bytes compressed)",
			filename, originalSize, outputFile.Len())
	}
}

func (d *DB) GetMqttProfile(id string) (ret *models.MqttProfile, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get(mqttProfileKey(id))
		if v == nil {
			return fmt.Errorf("failed to find mqtt profile")
		}
		ret = &models.MqttProfile{}
		return proto.Unmarshal(v, ret)
	})
	if err == nil {
		if ret.AllowPublish == nil {
			ret.AllowPublish = []string{}
		}

		if ret.AllowSubscribe == nil {
			ret.AllowSubscribe = []string{}
		}
	}
	return
}

func (d *DB) UpdateMqttProfile(id string, update_fn func(old *models.MqttProfile) (*models.MqttProfile, error)) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := mqttProfileKey(id)
		v := b.Get(key)
		var old_obj *models.MqttProfile = nil
		if v != nil {
			old_obj = &models.MqttProfile{}
			err := proto.Unmarshal(v, old_obj)
			if err != nil {
				return err
			}
		}
		new_obj, err := update_fn(old_obj)
		if err != nil {
			return err
		}
		if new_obj == nil {
			// Make sure that no clients refer to this profile
			c := b.Cursor()
			prefix := []byte("mqtt-client:")
			for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
				p := &models.MqttClient{}
				err := proto.Unmarshal(v, p)
				if err == nil {
					if p.ProfileId == id {
						return fmt.Errorf("cannot delete MQTT profile %s, it is in use by client %s", id, p.Id)
					}
				}
			}
			return b.Delete(key)
		}
		serialized, err := proto.Marshal(new_obj)
		if err != nil {
			return err
		}
		return b.Put(key, serialized)
	})
}

func (d *DB) ListMqttProfiles() (ret []*models.MqttProfile) {
	d.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(BucketName).Cursor()
		prefix := []byte("mqtt-profile:")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			p := &models.MqttProfile{}
			err := proto.Unmarshal(v, p)
			if err == nil {
				ret = append(ret, p)
			}
		}
		return nil
	})
	return
}

func (d *DB) GetMqttClient(id string) (ret *models.MqttClient, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		v := b.Get(mqttClientKey(id))
		if v == nil {
			return fmt.Errorf("failed to find mqtt client")
		}
		ret = &models.MqttClient{}
		return proto.Unmarshal(v, ret)
	})
	return
}

func (d *DB) UpdateMqttClient(id string, update_fn func(old *models.MqttClient) (*models.MqttClient, error)) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketName)
		key := mqttClientKey(id)
		v := b.Get(key)
		var old_obj *models.MqttClient = nil
		if v != nil {
			old_obj = &models.MqttClient{}
			err := proto.Unmarshal(v, old_obj)
			if err != nil {
				return err
			}
		}
		new_obj, err := update_fn(old_obj)
		if err != nil {
			return err
		}
		if new_obj == nil {
			return b.Delete(key)
		}
		// Validate that the profile exists.
		v = b.Get(mqttProfileKey(new_obj.ProfileId))
		if v == nil {
			return fmt.Errorf("profile not found")
		}
		serialized, err := proto.Marshal(new_obj)
		if err != nil {
			return err
		}
		// If ID changed, delete the old key
		if new_obj.Id != id {
			err = b.Delete(key)
			if err != nil {
				return err
			}
		}
		return b.Put(mqttClientKey(new_obj.Id), serialized)
	})
}

func (d *DB) ListMqttClients() (ret []*models.MqttClient) {
	d.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(BucketName).Cursor()
		prefix := []byte("mqtt-client:")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			p := &models.MqttClient{}
			err := proto.Unmarshal(v, p)
			if err == nil {
				ret = append(ret, p)
			}
		}
		return nil
	})
	return
}

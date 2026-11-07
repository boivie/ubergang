// Package kiosk keeps track of sign-in requests from kiosks: devices that
// can't sign in by themselves, such as a wall-mounted tablet. A kiosk shows a
// code, which a signed-in user approves from another device, and the kiosk
// then gets a session that is only valid on a single backend.
//
// Requests are short-lived and only kept in memory. A restart cancels them,
// and the kiosks then make new ones.
package kiosk

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"slices"
	"sync"
	"time"
)

const (
	// How long a request can be approved, after which the kiosk shows a new
	// code.
	RequestTTL = 10 * time.Minute
	// How long an approved request waits for the kiosk to poll it.
	approvedTTL = time.Minute

	// The maximum number of pending requests from a single client: an IPv4
	// address or an IPv6 /64 network.
	MaxPerClient = 10
	// The maximum number of pending requests in total.
	MaxTotal = 1000

	// The maximum number of codes that a user can enter that don't match a
	// request, within RequestTTL. Prevents guessing codes.
	MaxFailedLookups = 10
)

var (
	ErrTooManyRequests = errors.New("too many pending kiosk sign-in requests")
	ErrNotFound        = errors.New("kiosk sign-in request not found")
	ErrAlreadyApproved = errors.New("kiosk sign-in request is already approved")
)

// Approval is how a user approved a request.
type Approval struct {
	UserID string
	// A name for the session, e.g. "Kitchen tablet". May be empty.
	Name string
	// How long the session is valid for. Zero means until it's revoked.
	Duration time.Duration
}

type Request struct {
	// A secret that only the kiosk knows, and polls the request with. Whoever
	// knows it gets the session once the request has been approved.
	ID string
	// What the kiosk shows, and what the approver enters.
	Code string
	// The backend host that the session will be restricted to.
	Host string
	// The URL on Host that the kiosk goes to when it's signed in.
	Redirect  string
	IP        string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
	// Set once the request has been approved.
	Approval *Approval
}

type Store struct {
	mu     sync.Mutex
	now    func() time.Time
	byID   map[string]*Request
	byCode map[string]*Request
	// When each user entered a code that didn't match a request.
	failedLookups map[string][]time.Time
}

func NewStore() *Store {
	return &Store{
		now:           time.Now,
		byID:          make(map[string]*Request),
		byCode:        make(map[string]*Request),
		failedLookups: make(map[string][]time.Time),
	}
}

// Create adds a pending request from a kiosk at `ip`, to sign in to `host`
// and then go to `redirect`.
func (s *Store) Create(host, redirect, ip, userAgent string) (Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpired()

	if len(s.byID) >= MaxTotal {
		return Request{}, ErrTooManyRequests
	}
	client := clientKey(ip)
	fromClient := 0
	for _, r := range s.byID {
		if clientKey(r.IP) == client {
			fromClient++
		}
	}
	if fromClient >= MaxPerClient {
		return Request{}, ErrTooManyRequests
	}

	code, err := s.uniqueCode()
	if err != nil {
		return Request{}, err
	}
	now := s.now()
	r := &Request{
		ID:        rand.Text(),
		Code:      code,
		Host:      host,
		Redirect:  redirect,
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: now,
		ExpiresAt: now.Add(RequestTTL),
	}
	s.byID[r.ID] = r
	s.byCode[r.Code] = r
	return *r, nil
}

// Get returns the request with the ID `id`, unless it has expired.
func (s *Store) Get(id string) (Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpired()
	r, ok := s.byID[id]
	if !ok {
		return Request{}, ErrNotFound
	}
	return *r, nil
}

// GetPendingByCode returns the request showing `code`, unless it has expired
// or already been approved, for the user `userID` to approve it. A user that
// enters too many codes that don't match gets ErrTooManyRequests for a while.
func (s *Store) GetPendingByCode(code, userID string) (Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpired()
	if len(s.failedLookups[userID]) >= MaxFailedLookups {
		return Request{}, ErrTooManyRequests
	}
	r, ok := s.byCode[code]
	if !ok || r.Approval != nil {
		s.failedLookups[userID] = append(s.failedLookups[userID], s.now())
		return Request{}, ErrNotFound
	}
	return *r, nil
}

// Approve marks the request with the ID `id` as approved. The kiosk then has
// a short while to poll for it.
func (s *Store) Approve(id string, approval Approval) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpired()
	r, ok := s.byID[id]
	if !ok {
		return ErrNotFound
	}
	if r.Approval != nil {
		return ErrAlreadyApproved
	}
	r.Approval = &approval
	// The code can't be used anymore, so it can be shown by another kiosk.
	delete(s.byCode, r.Code)
	r.ExpiresAt = s.now().Add(approvedTTL)
	return nil
}

// TakeApproved removes and returns the request with the ID `id`, if it has
// been approved. A request can only be taken once.
func (s *Store) TakeApproved(id string) (Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpired()
	r, ok := s.byID[id]
	if !ok || r.Approval == nil {
		return Request{}, ErrNotFound
	}
	delete(s.byID, id)
	return *r, nil
}

func (s *Store) removeExpired() {
	now := s.now()
	for id, r := range s.byID {
		if !now.Before(r.ExpiresAt) {
			delete(s.byID, id)
			if s.byCode[r.Code] == r {
				delete(s.byCode, r.Code)
			}
		}
	}
	for userID, failures := range s.failedLookups {
		failures = slices.DeleteFunc(failures, func(t time.Time) bool {
			return !now.Before(t.Add(RequestTTL))
		})
		if len(failures) == 0 {
			delete(s.failedLookups, userID)
		} else {
			s.failedLookups[userID] = failures
		}
	}
}

func (s *Store) uniqueCode() (string, error) {
	// With at most MaxTotal codes in use, a collision is unlikely.
	for range 10 {
		n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
		if err != nil {
			return "", err
		}
		code := fmt.Sprintf("%06d", n)
		if _, taken := s.byCode[code]; !taken {
			return code, nil
		}
	}
	return "", errors.New("failed to generate a unique code")
}

// clientKey returns what requests from `ip` are counted by: the address
// itself for IPv4, and its /64 network for IPv6, since a single client often
// has a whole /64.
func clientKey(ip string) string {
	addr := net.ParseIP(ip)
	if addr == nil {
		return ip
	}
	if v4 := addr.To4(); v4 != nil {
		return v4.String()
	}
	return addr.Mask(net.CIDRMask(64, 128)).String()
}

// SameNetwork returns whether two client IP addresses look like they're on
// the same network: the same IPv4 address (e.g. behind the same NAT) or the
// same IPv6 /64. Devices on the same network can still get different results,
// e.g. if one uses IPv4 and the other IPv6, so this is only a hint.
func SameNetwork(a, b string) bool {
	return net.ParseIP(a) != nil && net.ParseIP(b) != nil && clientKey(a) == clientKey(b)
}

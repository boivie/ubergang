package kiosk

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newStore returns a store with a clock that the test controls.
func newStore() (*Store, *time.Time) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := NewStore()
	s.now = func() time.Time { return now }
	return s, &now
}

func create(t *testing.T, s *Store, ip string) Request {
	t.Helper()
	r, err := s.Create("app.example.com", "https://app.example.com/", ip, "ua")
	require.NoError(t, err)
	return r
}

func TestCreateAndGet(t *testing.T) {
	s, _ := newStore()
	r := create(t, s, "192.0.2.1")
	assert.Len(t, r.Code, 6)
	assert.NotEmpty(t, r.ID)
	assert.NotEqual(t, r.ID, r.Code)

	got, err := s.Get(r.ID)
	require.NoError(t, err)
	assert.Equal(t, r.Code, got.Code)

	byCode, err := s.GetPendingByCode(r.Code, "approver")
	require.NoError(t, err)
	assert.Equal(t, r.ID, byCode.ID)
}

func TestRequestsExpire(t *testing.T) {
	s, now := newStore()
	r := create(t, s, "192.0.2.1")
	*now = now.Add(RequestTTL)

	_, err := s.Get(r.ID)
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = s.GetPendingByCode(r.Code, "approver")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.ErrorIs(t, s.Approve(r.ID, Approval{UserID: "u"}), ErrNotFound)
}

func TestApproveAndTake(t *testing.T) {
	s, _ := newStore()
	r := create(t, s, "192.0.2.1")

	_, err := s.TakeApproved(r.ID)
	assert.ErrorIs(t, err, ErrNotFound, "can't take a pending request")

	require.NoError(t, s.Approve(r.ID, Approval{UserID: "u", Name: "Kitchen"}))
	assert.ErrorIs(t, s.Approve(r.ID, Approval{UserID: "other"}), ErrAlreadyApproved)
	_, err = s.GetPendingByCode(r.Code, "approver")
	assert.ErrorIs(t, err, ErrNotFound, "an approved code can't be approved again")

	taken, err := s.TakeApproved(r.ID)
	require.NoError(t, err)
	assert.Equal(t, "u", taken.Approval.UserID)
	assert.Equal(t, "Kitchen", taken.Approval.Name)

	_, err = s.TakeApproved(r.ID)
	assert.ErrorIs(t, err, ErrNotFound, "can only be taken once")
}

func TestApprovedRequestsExpireQuickly(t *testing.T) {
	s, now := newStore()
	r := create(t, s, "192.0.2.1")
	require.NoError(t, s.Approve(r.ID, Approval{UserID: "u"}))
	*now = now.Add(approvedTTL)

	_, err := s.TakeApproved(r.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestLimitPerClient(t *testing.T) {
	s, now := newStore()
	for range MaxPerClient {
		create(t, s, "192.0.2.1")
	}
	_, err := s.Create("app.example.com", "https://app.example.com/", "192.0.2.1", "ua")
	assert.ErrorIs(t, err, ErrTooManyRequests)

	// Other clients aren't affected.
	create(t, s, "192.0.2.2")

	// Once they expire, new requests can be made.
	*now = now.Add(RequestTTL)
	create(t, s, "192.0.2.1")
}

func TestLimitPerClientCountsIPv6Networks(t *testing.T) {
	s, _ := newStore()
	for i := range MaxPerClient {
		create(t, s, fmt.Sprintf("2001:db8:1:2::%x", i+1))
	}
	_, err := s.Create("app.example.com", "https://app.example.com/", "2001:db8:1:2:ffff::1", "ua")
	assert.ErrorIs(t, err, ErrTooManyRequests)

	create(t, s, "2001:db8:1:3::1")
}

func TestGlobalLimit(t *testing.T) {
	s, _ := newStore()
	for i := range MaxTotal {
		create(t, s, fmt.Sprintf("10.%d.%d.1", i/256, i%256))
	}
	_, err := s.Create("app.example.com", "https://app.example.com/", "192.0.2.1", "ua")
	assert.ErrorIs(t, err, ErrTooManyRequests)
}

func TestLimitFailedLookups(t *testing.T) {
	s, now := newStore()
	r := create(t, s, "192.0.2.1")
	wrong := "000000"
	if r.Code == wrong {
		wrong = "000001"
	}
	for range MaxFailedLookups {
		_, err := s.GetPendingByCode(wrong, "guesser")
		assert.ErrorIs(t, err, ErrNotFound)
	}
	// Even the right code is rejected now.
	_, err := s.GetPendingByCode(r.Code, "guesser")
	assert.ErrorIs(t, err, ErrTooManyRequests)

	// Other users aren't affected.
	_, err = s.GetPendingByCode(r.Code, "approver")
	assert.NoError(t, err)

	*now = now.Add(RequestTTL)
	_, err = s.GetPendingByCode(wrong, "guesser")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSameNetwork(t *testing.T) {
	assert.True(t, SameNetwork("192.0.2.1", "192.0.2.1"))
	assert.False(t, SameNetwork("192.0.2.1", "192.0.2.2"))
	assert.True(t, SameNetwork("2001:db8::1", "2001:db8::2"))
	assert.False(t, SameNetwork("2001:db8:0:1::1", "2001:db8:0:2::1"))
	assert.False(t, SameNetwork("192.0.2.1", "2001:db8::1"))
	assert.False(t, SameNetwork("", ""))
}

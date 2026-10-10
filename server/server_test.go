package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpsRedirectTarget(t *testing.T) {
	testCases := []struct {
		name        string
		requestHost string
		httpsPort   int
		expected    string
	}{
		{
			name:        "host with non-standard HTTP port is redirected to the HTTPS port",
			requestHost: "app.test.boivie.com:10080",
			httpsPort:   10443,
			expected:    "app.test.boivie.com:10443",
		},
		{
			name:        "explicit default HTTP port is replaced by the HTTPS port",
			requestHost: "app.test.boivie.com:80",
			httpsPort:   10443,
			expected:    "app.test.boivie.com:10443",
		},
		{
			name:        "explicit port is replaced by the default HTTPS port",
			requestHost: "app.home.boivie.com:80",
			httpsPort:   443,
			expected:    "app.home.boivie.com",
		},
		{
			name:        "host without a port is passed through unchanged",
			requestHost: "app.home.boivie.com",
			httpsPort:   443,
			expected:    "app.home.boivie.com",
		},
		{
			name:        "host without a port is passed through even for a non-standard HTTPS port",
			requestHost: "app.home.boivie.com",
			httpsPort:   10443,
			expected:    "app.home.boivie.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, httpsRedirectTarget(tc.requestHost, tc.httpsPort))
		})
	}
}

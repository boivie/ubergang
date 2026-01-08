package common

import (
	"net/http"
	"testing"
)

func TestReadUserIP(t *testing.T) {
	testCases := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "No headers, fallback to RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "192.0.2.1",
		},
		{
			name: "CF-Connecting-IP header",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.1",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.1",
		},
		{
			name: "True-Client-IP header",
			headers: map[string]string{
				"True-Client-IP": "203.0.113.2",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.2",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.3",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.3",
		},
		{
			name: "X-Forwarded-For header with a single IP",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.4",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.4",
		},
		{
			name: "X-Forwarded-For header with multiple IPs",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.5, 198.51.100.1, 192.0.2.1",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.5",
		},
		{
			name: "X-Forwarded-For header with IP and port",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.6:54321, 198.51.100.1, 192.0.2.1",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.6",
		},
		{
			name: "Forwarded header",
			headers: map[string]string{
				"Forwarded": "for=203.0.113.7;proto=http;by=192.0.2.1",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.7",
		},
		{
			name: "Forwarded header with multiple values",
			headers: map[string]string{
				"Forwarded": "for=203.0.113.8, for=198.51.100.2",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.8",
		},
		{
			name: "Header precedence",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.1",
				"X-Forwarded-For":  "203.0.113.4",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.1",
		},
		{
			name: "Invalid IP in headers",
			headers: map[string]string{
				"CF-Connecting-IP": "not-an-ip",
				"X-Forwarded-For":  "203.0.113.4",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.4",
		},
		{
			name: "Empty header values",
			headers: map[string]string{
				"CF-Connecting-IP": "",
				"X-Forwarded-For":  "203.0.113.4",
			},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.4",
		},
		{
			name:       "RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "192.0.2.2",
			expectedIP: "192.0.2.2",
		},
		{
			name:       "No valid IP found",
			headers:    map[string]string{},
			remoteAddr: "invalid-remote-addr",
			expectedIP: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{
				Header:     make(http.Header),
				RemoteAddr: tc.remoteAddr,
			}
			for key, value := range tc.headers {
				r.Header.Set(key, value)
			}

			ip := ReadUserIP(r)
			if ip != tc.expectedIP {
				t.Errorf("Expected IP %s, but got %s", tc.expectedIP, ip)
			}
		})
	}
}

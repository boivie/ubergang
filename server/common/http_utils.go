package common

import (
	"net"
	"net/http"
	"strings"
)

// ReadUserIP gets the user's IP address from the request.
// It checks a series of standard and non-standard headers in a specific order
// to identify the IP, which is important when the server is behind one or more
// reverse proxies.
func ReadUserIP(r *http.Request) string {
	// Headers are checked in a specific order of precedence.
	// Headers set by trusted proxies (like Cloudflare) should be checked first.
	// The `Forwarded` header is the IETF standard, but `X-Forwarded-For` is more common.
	headers := []string{
		"CF-Connecting-IP",
		"True-Client-IP",
		"X-Forwarded-For",
		"X-Real-IP",
		"Forwarded",
	}

	for _, header := range headers {
		headerValue := r.Header.Get(header)
		if headerValue == "" {
			continue
		}

		switch header {
		case "X-Forwarded-For":
			// Can be a comma-separated list of IPs. The first one is the client.
			// e.g., "client, proxy1, proxy2"
			ips := strings.Split(headerValue, ",")
			clientIP := strings.TrimSpace(ips[0])
			// The IP can have a port.
			host, _, err := net.SplitHostPort(clientIP)
			if err == nil {
				clientIP = host
			}
			if ip := net.ParseIP(clientIP); ip != nil {
				return ip.String()
			}

		case "Forwarded":
			// Can be a comma-separated list. The first one is the client.
			// e.g. Forwarded: for=192.0.2.43, for=198.51.100.17
			if forwardedParts := strings.Split(headerValue, ","); len(forwardedParts) > 0 {
				firstForwarded := strings.TrimSpace(forwardedParts[0])
				// e.g. for=192.0.2.60;proto=http;by=203.0.113.43
				for _, part := range strings.Split(firstForwarded, ";") {
					if strings.HasPrefix(strings.ToLower(strings.TrimSpace(part)), "for=") {
						value := strings.TrimPrefix(strings.ToLower(part), "for=")
						value = strings.Trim(value, `"`) // The value can be quoted
						if ip := net.ParseIP(value); ip != nil {
							return ip.String()
						}
					}
				}
			}

		default:
			// For other headers, we expect a single IP.
			if ip := net.ParseIP(headerValue); ip != nil {
				return ip.String()
			}
		}
	}

	// If no header is found or is valid, fall back to RemoteAddr.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}

	// r.RemoteAddr might not have a port, in which case SplitHostPort fails.
	// In that case, r.RemoteAddr is likely the IP address.
	if ip := net.ParseIP(r.RemoteAddr); ip != nil {
		return ip.String()
	}

	return "" // Return empty string if no valid IP is found
}

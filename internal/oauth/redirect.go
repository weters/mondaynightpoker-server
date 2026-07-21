package oauth

import "net/url"

// URI scheme constants for redirect URI validation.
const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

// isLoopbackHost reports whether host is an OAuth 2.1 loopback host.
func isLoopbackHost(host string) bool {
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// isValidRegistrationURI reports whether raw is an acceptable redirect URI for dynamic
// client registration: https on any host, or http on a loopback host.
func isValidRegistrationURI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	switch u.Scheme {
	case schemeHTTPS:
		return u.Host != ""
	case schemeHTTP:
		return isLoopbackHost(u.Hostname())
	default:
		return false
	}
}

// loopbackMatch reports whether provided matches a registered loopback redirect URI,
// ignoring the port per OAuth 2.1 section 8.4.2.
func loopbackMatch(registered, provided string) bool {
	r, err := url.Parse(registered)
	if err != nil {
		return false
	}

	p, err := url.Parse(provided)
	if err != nil {
		return false
	}

	if r.Scheme != schemeHTTP || p.Scheme != schemeHTTP {
		return false
	}

	if !isLoopbackHost(r.Hostname()) || !isLoopbackHost(p.Hostname()) {
		return false
	}

	return r.Path == p.Path
}

// redirectURIMatches reports whether provided exactly matches one of the registered
// redirect URIs, or matches a registered loopback URI on any port.
func redirectURIMatches(registered []string, provided string) bool {
	for _, reg := range registered {
		if reg == provided {
			return true
		}

		if loopbackMatch(reg, provided) {
			return true
		}
	}

	return false
}

package oauth

import (
	"net"
	"net/url"
	"strconv"
)

// URI scheme constants for redirect URI validation.
const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

// Loopback hosts permitted by OAuth 2.1 section 8.4.2.
const (
	hostIPv4Loopback = "127.0.0.1"
	hostIPv6Loopback = "::1"
	hostLocalhost    = "localhost"
)

// maxPort is the highest valid TCP port.
const maxPort = 65535

// loopbackHostLiteral maps an OAuth 2.1 loopback hostname onto its own constant. Returning
// a literal rather than the argument keeps request-controlled text out of any URI rebuilt
// from the result.
func loopbackHostLiteral(hostname string) (string, bool) {
	switch hostname {
	case hostIPv4Loopback:
		return hostIPv4Loopback, true
	case hostIPv6Loopback:
		return hostIPv6Loopback, true
	case hostLocalhost:
		return hostLocalhost, true
	}

	return "", false
}

// isLoopbackHost reports whether host is an OAuth 2.1 loopback host.
func isLoopbackHost(host string) bool {
	_, ok := loopbackHostLiteral(host)
	return ok
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

// canonicalPort re-renders a port from its parsed numeric form, rejecting anything that is
// not a valid TCP port. An empty port stays empty, meaning the scheme default.
func canonicalPort(port string) (string, bool) {
	if port == "" {
		return "", true
	}

	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > maxPort {
		return "", false
	}

	return strconv.Itoa(n), true
}

// loopbackRedirect reports whether provided matches a registered loopback redirect URI,
// ignoring the port per OAuth 2.1 section 8.4.2, and returns the URI to redirect to. The
// result is rebuilt from the registered entry plus the request's (revalidated) loopback
// host and port, so the caller never echoes the raw request value back to the browser.
func loopbackRedirect(registered, provided string) (string, bool) {
	r, err := url.Parse(registered)
	if err != nil {
		return "", false
	}

	p, err := url.Parse(provided)
	if err != nil {
		return "", false
	}

	if r.Scheme != schemeHTTP || p.Scheme != schemeHTTP {
		return "", false
	}

	if !isLoopbackHost(r.Hostname()) || r.Path != p.Path {
		return "", false
	}

	host, ok := loopbackHostLiteral(p.Hostname())
	if !ok {
		return "", false
	}

	port, ok := canonicalPort(p.Port())
	if !ok {
		return "", false
	}

	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if host == hostIPv6Loopback {
		host = "[" + host + "]"
	}

	u := url.URL{Scheme: schemeHTTP, Host: host, Path: r.Path, RawQuery: r.RawQuery}

	return u.String(), true
}

// resolveRedirectURI validates provided against the client's registered redirect URIs and
// returns the URI the browser should be sent to. The returned value is always derived from
// the registered entry (a loopback client additionally contributes its port), never from
// the request, so it is safe to use as a redirect target.
func resolveRedirectURI(registered []string, provided string) (string, bool) {
	if provided == "" {
		return "", false
	}

	for _, reg := range registered {
		if reg == provided {
			return reg, true
		}

		if uri, ok := loopbackRedirect(reg, provided); ok {
			return uri, true
		}
	}

	return "", false
}

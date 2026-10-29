package controller

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// permissiveJar is an http.CookieJar that accepts cookies for plain
// IP-address hosts, which Go's default cookiejar rejects. Omada controllers
// are commonly reached by IP address, and the login session cookie is only
// useful if it is stored for that host.
type permissiveJar struct {
	mu     sync.Mutex
	byHost map[string][]*http.Cookie
}

func newPermissiveJar() http.CookieJar {
	return &permissiveJar{byHost: make(map[string][]*http.Cookie)}
}

func (j *permissiveJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()

	host := j.key(u)
	now := time.Now()
	existing := j.byHost[host]

	for _, c := range cookies {
		if !c.Expires.IsZero() && !c.Expires.After(now) {
			// An expired cookie removes the stored entry with the same name+path.
			kept := existing[:0]
			for _, e := range existing {
				if e.Name == c.Name && e.Path == c.Path {
					continue
				}
				kept = append(kept, e)
			}
			existing = kept
			continue
		}
		replaced := false
		for i, e := range existing {
			if e.Name == c.Name && e.Path == c.Path {
				existing[i] = c
				replaced = true
				break
			}
		}
		if !replaced {
			existing = append(existing, c)
		}
	}
	j.byHost[host] = existing
}

func (j *permissiveJar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()

	host := j.key(u)
	var out []*http.Cookie
	for _, c := range j.byHost[host] {
		if strings.HasPrefix(u.Path, c.Path) {
			out = append(out, c)
		}
	}
	return out
}

func (j *permissiveJar) key(u *url.URL) string {
	return u.Host
}

// ClearDomain removes all cookies stored for the given host.
func (j *permissiveJar) ClearDomain(host string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	delete(j.byHost, host)
}

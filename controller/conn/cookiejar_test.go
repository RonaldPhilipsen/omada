package conn

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestJarAcceptsIPHost(t *testing.T) {
	jar := newPermissiveJar()
	u, _ := url.Parse("https://192.168.1.10/login")
	jar.SetCookies(u, []*http.Cookie{{Name: "SESSION", Value: "abc", Path: "/"}})
	got := jar.Cookies(u)
	if len(got) != 1 || got[0].Value != "abc" {
		t.Errorf("got %v", got)
	}
}

func TestJarPathScoping(t *testing.T) {
	jar := newPermissiveJar()
	base, _ := url.Parse("https://omada.local")
	api, _ := url.Parse("https://omada.local/api/login")
	jar.SetCookies(api, []*http.Cookie{{Name: "SESSION", Value: "abc", Path: "/api"}})

	other, _ := url.Parse("https://omada.local/other")
	if got := jar.Cookies(other); len(got) != 0 {
		t.Errorf("cookie with path /api leaked to /other: %v", got)
	}
	if got := jar.Cookies(api); len(got) != 1 {
		t.Errorf("expected 1 cookie for /api, got %v", got)
	}
	_ = base
}

func TestJarReplaceByName(t *testing.T) {
	jar := newPermissiveJar()
	u, _ := url.Parse("https://192.168.1.10/login")
	jar.SetCookies(u, []*http.Cookie{{Name: "SESSION", Value: "old", Path: "/"}})
	jar.SetCookies(u, []*http.Cookie{{Name: "SESSION", Value: "new", Path: "/"}})
	got := jar.Cookies(u)
	if len(got) != 1 || got[0].Value != "new" {
		t.Errorf("expected replacement, got %v", got)
	}

	// Expired cookies are dropped
	jar.SetCookies(u, []*http.Cookie{{Name: "SESSION", Value: "dead", Path: "/", Expires: time.Now().Add(-time.Hour)}})
	if got := jar.Cookies(u); len(got) != 0 {
		t.Errorf("expired cookie not dropped: %v", got)
	}
}

func TestJarHostScoping(t *testing.T) {
	jar := newPermissiveJar()
	a, _ := url.Parse("https://192.168.1.10/login")
	b, _ := url.Parse("https://omada.local/login")
	jar.SetCookies(a, []*http.Cookie{{Name: "SESSION", Value: "a", Path: "/"}})
	if got := jar.Cookies(b); len(got) != 0 {
		t.Errorf("cookie leaked across hosts: %v", got)
	}
}

func TestNewConnNormalizesURL(t *testing.T) {
	c, err := newConn("192.168.1.10", "u", "p", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if c.base != "https://192.168.1.10" || c.host != "192.168.1.10" {
		t.Errorf("got base=%q host=%q", c.base, c.host)
	}

	c2, err2 := newConn("http://omada.local:8043/", "u", "p", &http.Client{})
	if err2 != nil {
		t.Fatal(err2)
	}
	if c2.base != "http://omada.local:8043" {
		t.Errorf("got base=%q", c2.base)
	}
	if c2.host != "omada.local" {
		t.Errorf("got host=%q", c2.host)
	}

	if _, err := newConn("omada", "u", "p", nil); err == nil {
		t.Error("nil http client should be rejected")
	}
}

func TestConnURLs(t *testing.T) {
	c, _ := newConn("https://omada", "u", "p", &http.Client{})
	c.controllerID = "abc"

	if got := c.url("cmd/reboot", ""); got != "https://omada/abc/api/v2/cmd/reboot" {
		t.Errorf("url: %s", got)
	}
	if got := c.url("clients", "s1"); got != "https://omada/abc/api/v2/sites/s1/clients" {
		t.Errorf("url with site: %s", got)
	}
}

func TestNewConnKeepsHTTPScheme(t *testing.T) {
	c, err := newConn("http://192.168.1.10:8043", "u", "p", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if c.base != "http://192.168.1.10:8043" {
		t.Errorf("got base=%q", c.base)
	}
}

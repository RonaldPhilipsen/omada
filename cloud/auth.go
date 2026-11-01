package cloud

import "context"

// authenticator produces the full Authorization header value for a request
// and can drop its cached token so the next call re-authenticates. Both the
// hosted (token.go) and on-prem org-level (orgauth.go) token providers
// implement it; the shared authTransport consumes this interface.
type authenticator interface {
	HeaderValue(ctx context.Context) (string, error) // full Authorization header value
	Invalidate()                                     // drop the cached token
}

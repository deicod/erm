package oidc

import "context"

// Claims captures identity metadata extracted from verified tokens.
type Claims struct {
	Subject       string
	Email         string
	Name          string
	Username      string
	GivenName     string
	FamilyName    string
	EmailVerified bool
	Roles         []string
	Raw           map[string]any
}

type claimsKey struct{}

// ToContext attaches claims to the context for downstream directives.
func ToContext(ctx context.Context, claims Claims) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, claimsKey{}, claims)
}

// FromContext extracts claims if present.
func FromContext(ctx context.Context) (Claims, bool) {
	if ctx == nil {
		return Claims{}, false
	}
	claims, ok := ctx.Value(claimsKey{}).(Claims)
	return claims, ok
}

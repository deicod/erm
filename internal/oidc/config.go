package oidc

import (
	"context"
	"errors"
	"net/http"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
)

// Config describes how to bootstrap the OIDC middleware.
type Config struct {
	Issuer     string
	Audiences  []string
	HTTPClient *http.Client
	Mapper     ClaimsMapper
	// Skip expiry or issuer checks are primarily for tests.
	SkipExpiryCheck bool
	SkipIssuerCheck bool
}

// NewMiddleware builds a Middleware by performing OIDC discovery and wiring the ID token verifier.
func NewMiddleware(ctx context.Context, cfg Config) (*Middleware, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.Issuer == "" {
		return nil, errors.New("oidc: issuer required")
	}
	if len(cfg.Audiences) == 0 {
		return nil, errors.New("oidc: at least one audience required")
	}
	audiences := uniqueStrings(cfg.Audiences)
	if cfg.HTTPClient != nil {
		ctx = gooidc.ClientContext(ctx, cfg.HTTPClient)
	}
	provider, err := gooidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, err
	}

	oidcCfg := &gooidc.Config{
		SkipExpiryCheck: cfg.SkipExpiryCheck,
		SkipIssuerCheck: cfg.SkipIssuerCheck,
	}
	if len(audiences) == 1 && !cfg.SkipIssuerCheck {
		oidcCfg.ClientID = audiences[0]
	} else {
		oidcCfg.SkipClientIDCheck = true
	}

	verifier := provider.Verifier(oidcCfg)
	mapper := cfg.Mapper
	if mapper == nil {
		mapper = KeycloakClaimsMapper{}
	}

	return &Middleware{
		Verifier:  verifier,
		Mapper:    mapper,
		audiences: audiences,
	}, nil
}

func uniqueStrings(in []string) []string {
	if len(in) <= 1 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return in
	}
	return out
}

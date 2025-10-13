package oidc

import (
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type ClaimsMapper interface {
	Map(raw map[string]any) (Claims, error)
}

type Middleware struct {
	Verifier  *oidc.IDTokenVerifier
	Mapper    ClaimsMapper
	audiences []string
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}
		tokStr := strings.TrimPrefix(h, "Bearer ")
		// Verify via OIDC verifier
		idTok, err := m.Verifier.Verify(r.Context(), tokStr)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if len(m.audiences) > 0 && !audAllowed(idTok.Audience, m.audiences) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Parse claims as generic map
		var raw map[string]any
		if err := idTok.Claims(&raw); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Fallback: ensure subject present
		if _, ok := raw["sub"]; !ok {
			if t, err := jwt.Parse([]byte(tokStr), jwt.WithVerify(false)); err == nil {
				raw["sub"] = t.Subject()
			}
		}
		claims, err := m.Mapper.Map(raw)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := ToContext(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func audAllowed(actual []string, expected []string) bool {
	if len(expected) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, aud := range actual {
		set[aud] = struct{}{}
	}
	for _, allowed := range expected {
		if _, ok := set[allowed]; ok {
			return true
		}
	}
	return false
}

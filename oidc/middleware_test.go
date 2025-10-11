package oidc_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/deicod/erm/oidc"
)

type oidcTestEnv struct {
	t        *testing.T
	server   *httptest.Server
	audience string
	token    string
	client   *http.Client
}

func newOIDCTestEnv(t *testing.T) *oidcTestEnv {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privKey, err := jwk.FromRaw(key)
	if err != nil {
		t.Fatalf("jwk from raw: %v", err)
	}
	_ = privKey.Set(jwk.KeyUsageKey, "sig")
	_ = privKey.Set(jwk.AlgorithmKey, jwa.RS256)
	_ = privKey.Set(jwk.KeyIDKey, "test-key")

	pubKey, err := jwk.PublicKeyOf(privKey)
	if err != nil {
		t.Fatalf("public key: %v", err)
	}
	_ = pubKey.Set(jwk.KeyUsageKey, "sig")
	_ = pubKey.Set(jwk.AlgorithmKey, jwa.RS256)
	_ = pubKey.Set(jwk.KeyIDKey, "test-key")

	set := jwk.NewSet()
	set.AddKey(pubKey)

	env := &oidcTestEnv{
		t:        t,
		audience: "erm-client",
	}

	mux := http.NewServeMux()
	var issuer string
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		cfg := map[string]any{
			"issuer":   issuer,
			"jwks_uri": issuer + "/keys",
		}
		_ = json.NewEncoder(w).Encode(cfg)
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(set)
	})

	server := httptest.NewServer(mux)
	env.server = server
	env.client = server.Client()
	issuer = server.URL

	token := jwt.New()
	_ = token.Set(jwt.IssuerKey, issuer)
	_ = token.Set(jwt.AudienceKey, env.audience)
	_ = token.Set(jwt.SubjectKey, "user-123")
	_ = token.Set(jwt.IssuedAtKey, time.Now())
	_ = token.Set(jwt.ExpirationKey, time.Now().Add(5*time.Minute))
	_ = token.Set("email", "user@example.com")
	_ = token.Set("preferred_username", "jane")
	_ = token.Set("realm_access", map[string]any{"roles": []string{"admin", "user"}})

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, privKey))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	env.token = string(signed)
	return env
}

func (e *oidcTestEnv) Close() { e.server.Close() }

func TestMiddlewareSuccess(t *testing.T) {
	env := newOIDCTestEnv(t)
	defer env.Close()

	ctx := context.Background()
	mw, err := oidc.NewMiddleware(ctx, oidc.Config{
		Issuer:     env.server.URL,
		Audiences:  []string{env.audience},
		HTTPClient: env.client,
	})
	if err != nil {
		t.Fatalf("NewMiddleware: %v", err)
	}

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := oidc.FromContext(r.Context())
		if !ok {
			t.Fatalf("claims missing from context")
		}
		if claims.Username != "jane" {
			t.Fatalf("unexpected username %q", claims.Username)
		}
		if len(claims.Roles) != 2 {
			t.Fatalf("expected roles propagated, got %#v", claims.Roles)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestMiddlewareAudienceMismatch(t *testing.T) {
	env := newOIDCTestEnv(t)
	defer env.Close()

	mw, err := oidc.NewMiddleware(context.Background(), oidc.Config{
		Issuer:     env.server.URL,
		Audiences:  []string{"other-client"},
		HTTPClient: env.client,
	})
	if err != nil {
		t.Fatalf("NewMiddleware: %v", err)
	}

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be invoked on audience mismatch")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

package oidc

import "fmt"

type KeycloakClaimsMapper struct{}

func (KeycloakClaimsMapper) Map(raw map[string]any) (Claims, error) {
	get := func(k string) string { if v, ok := raw[k].(string); ok { return v }; return "" }
	roles := []string{}
	if ra, ok := raw["realm_access"].(map[string]any); ok {
		if rr, ok := ra["roles"].([]any); ok {
			for _, v := range rr { if s, ok := v.(string); ok { roles = append(roles, s) } }
		}
	}
	return Claims{
		Subject: get("sub"),
		Email: get("email"),
		Name: get("name"),
		Username: get("preferred_username"),
		GivenName: get("given_name"),
		FamilyName: get("family_name"),
		EmailVerified: raw["email_verified"] == true,
		Roles: roles,
		Raw: raw,
	}, nil
}

func ValidateConfig(issuer, audience string) error {
	if issuer == "" || audience == "" { return fmt.Errorf("issuer and audience required") }
	return nil
}

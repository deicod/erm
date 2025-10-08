module github.com/deicod/erm

go 1.25.2

require (
    github.com/99designs/gqlgen v0.17.80 // indirect until wired by user app
    github.com/coreos/go-oidc/v3 v3.16.0
    github.com/google/uuid v1.6.0
    github.com/jackc/pgx/v5 v5.7.6
    github.com/lestrrat-go/jwx/v3 v3.0.11
    github.com/spf13/cobra v1.10.1
)

replace github.com/deicod/erm => .

// Package testkit provides helpers for exercising generated ORM and GraphQL layers in tests.
//
// The utilities intentionally avoid network calls so unit and integration tests can run quickly
// within CI. See sandbox.go and graphql.go for usage examples.
package testkit

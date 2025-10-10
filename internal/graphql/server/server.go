package server

import (
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// NewServer configures a gqlgen handler with HTTP and subscription transports.
func NewServer(opts Options) *handler.Server {
	opts = normaliseOptions(opts)
	srv := handler.New(NewExecutableSchema(opts))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	if opts.Subscriptions.Enabled {
		if opts.Subscriptions.Transports.Websocket {
			srv.AddTransport(&transport.Websocket{KeepAlivePingInterval: 15 * time.Second})
		}
		if opts.Subscriptions.Transports.GraphQLWS {
			srv.AddTransport(&transport.Websocket{KeepAlivePingInterval: 15 * time.Second, InitTimeout: 30 * time.Second})
		}
	}
	return srv
}

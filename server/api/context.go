package api

import (
	"context"
	"net"
)

type contextKey int

const (
	httpConnContextKey contextKey = iota
	sessionContextKey
)

// SetContextConn stores the connection in the request context.
func SetContextConn(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, httpConnContextKey, c)
}

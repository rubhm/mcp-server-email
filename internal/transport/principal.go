package transport

import "context"

type principalKey struct{}

// Principal is an instance identity, not an Open WebUI user or a permission grant.
type Principal struct {
	ID        string
	Transport string
}

// PrincipalFromContext returns only identities installed by the transport boundary.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)

	return principal, ok
}

func authenticatedContext(ctx context.Context, mode string) context.Context {
	return context.WithValue(ctx, principalKey{}, Principal{ID: "instance", Transport: mode})
}

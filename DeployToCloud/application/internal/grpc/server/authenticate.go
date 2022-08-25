package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type subjectContextKey struct{}

func authenticate(ctx context.Context) (context.Context, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(codes.Unknown, "couldn't find peer info").Err()
	}

	if p.AuthInfo == nil {
		return ctx, status.New(codes.Unauthenticated, "no transport security being used").Err()
	}

	t, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ctx, status.New(codes.Unauthenticated, "failed to cast AuthInfo").Err()
	}
	subject := t.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

func subject(ctx context.Context) string {
	//nolint:forcetypeassert //reason: no problem
	return ctx.Value(subjectContextKey{}).(string)
}

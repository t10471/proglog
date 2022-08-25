package auth

import (
	"fmt"

	"github.com/casbin/casbin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type IAuthorizer interface {
	Authorize(subject, object, action string) error
}

type Authorizer struct {
	enforcer *casbin.Enforcer
}

type Args struct {
	ModelFile  string
	PolicyFile string
}

func NewAuthorizer(args Args) *Authorizer {
	enforcer := casbin.NewEnforcer(args.ModelFile, args.PolicyFile)
	return &Authorizer{enforcer: enforcer}
}

func (a *Authorizer) Authorize(subject, object, action string) error {
	if a.enforcer.Enforce(subject, object, action) {
		return nil
	}
	msg := fmt.Sprintf("%s not permitted to %s to %s", subject, action, object)
	st := status.New(codes.PermissionDenied, msg)
	return st.Err()
}

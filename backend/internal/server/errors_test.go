package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSandboxErrorsReachAppsWithoutWrapping(t *testing.T) {
	sandbox := status.Error(codes.Internal, "INTERNAL: that file is an app, not an extension")
	wrapped := &gqlerror.Error{
		Err:       fmt.Errorf("inspecting extension: %w", sandbox),
		Message:   "inspecting extension: " + sandbox.Error(),
		Path:      ast.Path{ast.PathName("installExternalExtension")},
		Locations: []gqlerror.Location{{Line: 2, Column: 5}},
	}
	ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, graphql.DefaultRecover)
	got := sandboxErrorPresenter(ctx, wrapped)
	if got.Message != "that file is an app, not an extension" {
		t.Fatalf("message %q", got.Message)
	}
	if got.Extensions["code"] != "INTERNAL" {
		t.Fatalf("code %v", got.Extensions["code"])
	}
}

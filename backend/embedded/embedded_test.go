package embedded

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/grpc"

	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
)

type host struct {
	sandboxv1.UnimplementedExtensionServiceServer
}

func (host) ListLoadedExtensions(context.Context, *sandboxv1.Empty) (*sandboxv1.ExtensionList, error) {
	return &sandboxv1.ExtensionList{}, nil
}

func TestStartRequiresToken(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	g := grpc.NewServer()
	sandboxv1.RegisterExtensionServiceServer(g, host{})
	go g.Serve(lis)
	defer g.Stop()

	if err := Start(t.TempDir(), lis.Addr().String(), "host-token"); err != nil {
		t.Fatal(err)
	}
	defer Stop()
	if err := Start(t.TempDir(), lis.Addr().String(), "x"); err == nil {
		t.Fatal("second Start succeeded")
	}

	query := func(token string) int {
		req, _ := http.NewRequest(http.MethodPost, BaseURL()+"/api/graphql", strings.NewReader(`{"query":"{ __typename }"}`))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if code := query(""); code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", code)
	}
	if code := query(APIToken()); code != http.StatusOK {
		t.Fatalf("with token: %d", code)
	}

	if err := Stop(); err != nil {
		t.Fatal(err)
	}
	if BaseURL() != "" {
		t.Fatal("BaseURL set after Stop")
	}
}

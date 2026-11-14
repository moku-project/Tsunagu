package sandbox

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
)

type bigImageServer struct {
	sandboxv1.UnimplementedExtensionServiceServer
	size int
}

func (s *bigImageServer) GetImageBytes(context.Context, *sandboxv1.ImageRequest) (*sandboxv1.ImageData, error) {
	return &sandboxv1.ImageData{Data: make([]byte, s.size), ContentType: "image/jpeg"}, nil
}

// Long-strip pages are often well over gRPC's 4MB default message size.
func TestGetImageBytesAcceptsLargeImages(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	sandboxv1.RegisterExtensionServiceServer(srv, &bigImageServer{size: 24 << 20})
	go srv.Serve(lis)
	defer srv.Stop()

	c, err := NewClient(lis.Addr().String(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	img, err := c.GetImageBytes(context.Background(), "ext", "https://example.com/page.jpg")
	if err != nil {
		t.Fatalf("24MB image: %v", err)
	}
	if len(img.Data) != 24<<20 {
		t.Fatalf("got %d bytes, want %d", len(img.Data), 24<<20)
	}
}

type tokenServer struct {
	sandboxv1.UnimplementedExtensionServiceServer
	got chan []string
}

func (s *tokenServer) ListLoadedExtensions(ctx context.Context, _ *sandboxv1.Empty) (*sandboxv1.ExtensionList, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.got <- md.Get("authorization")
	return &sandboxv1.ExtensionList{}, nil
}

func TestExternalSandboxGetsToken(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ts := &tokenServer{got: make(chan []string, 4)}
	srv := grpc.NewServer()
	sandboxv1.RegisterExtensionServiceServer(srv, ts)
	go srv.Serve(lis)
	defer srv.Stop()

	sc := NewSupervised(SupervisedConfig{Addr: lis.Addr().String(), Token: "s3cret", External: true})
	defer sc.Shutdown()
	if _, err := sc.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := <-ts.got; len(got) != 1 || got[0] != "Bearer s3cret" {
		t.Fatalf("authorization = %v", got)
	}
}

func TestExternalSandboxAttachesLater(t *testing.T) {
	sc := NewSupervised(SupervisedConfig{External: true})
	defer sc.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := sc.Ensure(ctx); !errors.Is(err, ErrHostNotAttached) {
		t.Fatalf("Ensure before Attach = %v, want ErrHostNotAttached", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ts := &tokenServer{got: make(chan []string, 4)}
	srv := grpc.NewServer()
	sandboxv1.RegisterExtensionServiceServer(srv, ts)
	go srv.Serve(lis)
	defer srv.Stop()

	if err := sc.Attach(lis.Addr().String(), "later"); err != nil {
		t.Fatal(err)
	}
	if _, err := sc.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := <-ts.got; len(got) != 1 || got[0] != "Bearer later" {
		t.Fatalf("authorization = %v", got)
	}
}

func TestAttachRejectsSpawnedSandbox(t *testing.T) {
	sc := NewSupervised(SupervisedConfig{Addr: "127.0.0.1:1"})
	defer sc.Shutdown()
	if err := sc.Attach("127.0.0.1:2", "x"); err == nil {
		t.Fatal("Attach succeeded on a sandbox this process spawns")
	}
}

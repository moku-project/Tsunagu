package sandbox

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"

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

	c, err := NewClient(lis.Addr().String())
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

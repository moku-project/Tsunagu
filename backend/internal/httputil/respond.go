package httputil

import (
	"encoding/json"
	"log"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func JSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func Error(w http.ResponseWriter, message string, code int) {
	http.Error(w, message, code)
}

func GRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	var code int
	switch st.Code() {
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.NotFound:
		code = http.StatusNotFound
	case codes.AlreadyExists:
		code = http.StatusConflict
	case codes.PermissionDenied, codes.Unauthenticated:
		code = http.StatusForbidden
	case codes.DeadlineExceeded:
		code = http.StatusGatewayTimeout
	case codes.Unavailable:
		code = http.StatusServiceUnavailable
	default:
		code = http.StatusBadGateway
	}

	Error(w, st.Message(), code)
}

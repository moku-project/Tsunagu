FROM golang:1.23-alpine AS build
RUN apk add --no-cache build-base vips-dev pkgconfig
WORKDIR /src
COPY backend/go.mod ./
RUN go mod download || true
COPY backend/ ./
RUN go build -o /out/tsunagu ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates vips
COPY --from=build /out/tsunagu /usr/local/bin/tsunagu
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/tsunagu"]

# HTTP mode only — exposes /search, /url, /health (see internal/server).
# Build from music-hub: context ./projects/muze

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X github.com/ropean/muze/internal/selfupdate.Version=${VERSION}" \
    -o /out/muze .

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
COPY --from=build /out/muze /usr/local/bin/muze
EXPOSE 8010
ENTRYPOINT ["/usr/local/bin/muze"]
CMD ["serve", "--port", "8010"]

# HTTP mode only — exposes /search, /url, /health (see internal/server).
# Build from music-hub: context ./projects/music-provider-cn

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/music-provider-cn .

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
COPY --from=build /out/music-provider-cn /usr/local/bin/music-provider-cn
EXPOSE 8010
ENTRYPOINT ["/usr/local/bin/music-provider-cn"]
CMD ["serve", "--port", "8010"]

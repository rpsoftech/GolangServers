# pm2go hub (web dashboard + API). Build from the repository root:
#   docker build -f servers/pm2go/deploy/Dockerfile -t pm2go-hub .
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY servers/pm2go ./servers/pm2go
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X github.com/rpsoftech/golang-servers/servers/pm2go/cli.Version=${VERSION}" \
    -o /out/pm2go ./servers/pm2go

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget \
 && addgroup -S pm2go && adduser -S -G pm2go -h /data pm2go \
 && mkdir -p /data && chown pm2go:pm2go /data
COPY --from=build /out/pm2go /usr/local/bin/pm2go
USER pm2go
ENV PMGO_HUB_HOME=/data PMGO_TRUST_PROXY=1
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["pm2go", "hub"]
CMD ["--listen", ":8080"]

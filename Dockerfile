# syntax=docker/dockerfile:1

## Build Stage
FROM golang:1.25.1-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags "-w -s -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o /build/docker-hub-cleaner \
    ./cmd/docker-hub-cleaner

## Runtime Stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
RUN addgroup   -g 1000    cleaner && \
    adduser -D -u 1000 -G cleaner cleaner
COPY --from=builder /build/docker-hub-cleaner /usr/local/bin/docker-hub-cleaner
USER cleaner
ENTRYPOINT ["docker-hub-cleaner"]
CMD ["--help"]

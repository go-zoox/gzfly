# Builder
FROM golang:1.20-alpine as builder
RUN apk add --no-cache gcc g++ make git
WORKDIR /build
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux \
  GOARCH=amd64 \
  go build \
  -trimpath \
  -ldflags '-w -s -buildid=' \
  -v -o gzfly

# Server
FROM golang:1.20-alpine
LABEL MAINTAINER="Zero<tobewhatwewant@gmail.com>"
LABEL org.opencontainers.image.source="https://github.com/go-zoox/gzfly"
ARG VERSION=latest
ENV MODE=production
COPY --from=builder /build/gzfly /bin
ENV VERSION=${VERSION}
CMD gzfly server -c /conf/config.yml

# Builder
FROM --platform=$BUILDPLATFORM whatwewant/builder-go:v1.20-1 as builder

WORKDIR /build

COPY go.mod ./

COPY go.sum ./

RUN go mod download

COPY . .

ARG TARGETARCH

RUN CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=$TARGETARCH \
  go build \
  -trimpath \
  -ldflags '-w -s -buildid=' \
  -v -o gzfly

# Server
FROM whatwewant/go:v1.20-1

LABEL MAINTAINER="Zero<tobewhatwewant@gmail.com>"

LABEL org.opencontainers.image.source="https://github.com/go-zoox/gzfly"

ARG VERSION=latest

ENV MODE=production

COPY --from=builder /build/gzfly /bin

ENV VERSION=${VERSION}

CMD gzfly server -c /conf/config.yml

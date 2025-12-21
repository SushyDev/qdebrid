# --- Build app
FROM nixos/nix:latest AS app

RUN mkdir -p /root/.config/nix && \
    echo "experimental-features = nix-command flakes" > /root/.config/nix/nix.conf

RUN nix profile add nixpkgs#go

ENV GO111MODULE=on \
    GOPROXY=direct \
    GOFLAGS=-mod=readonly \
    GOTOOLCHAIN=go1.25.4+auto

WORKDIR /src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -extldflags '-static'" -o /out/main ./cmd/qdebrid

# --- Construct final image
FROM alpine:3.21

# Install ffmpeg (which includes ffprobe)
RUN apk add --no-cache ffmpeg ca-certificates

ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

ENV PATH=/bin:/usr/bin

WORKDIR /app

COPY --from=app /out/main /bin/main

ENTRYPOINT ["/bin/main"]

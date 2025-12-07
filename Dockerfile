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

# --- Build dependencies
FROM nixos/nix:latest AS dependencies

RUN mkdir -p /root/.config/nix && \
    echo "experimental-features = nix-command flakes" > /root/.config/nix/nix.conf

WORKDIR /src

COPY nix ./

RUN nix build ./ --out-link /out

# --- Construct final image
FROM scratch

ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

ENV PATH=/bin

WORKDIR /app

COPY --from=app /out/main /bin/main
COPY --from=dependencies /out/etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/bin/main"]

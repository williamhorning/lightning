FROM --platform=$BUILDPLATFORM golang@sha256:ac09a5f469f307e5da71e766b0bd59c9c49ea460a528cc3e6686513d64a6f1fb AS builder

ARG VERSION
ARG SOURCE_DATE_STR

WORKDIR /src

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
      -tags goolm -trimpath -buildvcs=false -ldflags="-buildid=" -o /out/lightning ./cmd/lightning/ && \
      touch -d "$SOURCE_DATE_STR" /out/lightning

FROM scratch

ARG VERSION

LABEL \
  maintainer="William Horning" \
  org.opencontainers.image.title="Lightning" \
  org.opencontainers.image.description="extensible chatbot connecting communities" \
  org.opencontainers.image.version=$VERSION \
  org.opencontainers.image.source="https://codeberg.org/jersey/lightning" \
  org.opencontainers.image.licenses="MIT"

USER 1001:1001

COPY --from=builder /out/lightning /lightning
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

VOLUME ["/data"]
WORKDIR /data

ENTRYPOINT ["/lightning"]

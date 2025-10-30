FROM golang:1.25.3-alpine AS builder

COPY ./pkg/platforms/stoat/cert.pem /usr/local/share/ca-certificates
RUN update-ca-certificates

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -tags goolm -o /out/lightning ./cmd/lightning/main.go

FROM scratch

LABEL maintainer="William Horning"
LABEL version="0.8.0-rc.6"
LABEL description="Lightning"
LABEL org.opencontainers.image.title="Lightning"
LABEL org.opencontainers.image.description="extensible chatbot connecting communities"
LABEL org.opencontainers.image.version="0.8.0-rc.6"
LABEL org.opencontainers.image.source="https://github.com/williamhorning/lightning"
LABEL org.opencontainers.image.licenses="MIT"

USER 1001:1001
COPY --from=builder /out/lightning /lightning
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
VOLUME [ "/data" ]
WORKDIR /data

ENTRYPOINT ["/lightning"]

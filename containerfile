FROM golang:1.25.0-alpine AS builder

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -tags goolm -o /out/lightning ./cmd/lightning/main.go

FROM scratch

# metadata
LABEL maintainer="William Horning"
LABEL version="0.8.0-rc.2"
LABEL description="Lightning"
LABEL org.opencontainers.image.title="Lightning"
LABEL org.opencontainers.image.description="extensible chatbot connecting communities"
LABEL org.opencontainers.image.version="0.8.0-rc.2"
LABEL org.opencontainers.image.source="https://github.com/williamhorning/lightning"
LABEL org.opencontainers.image.licenses="MIT"

# copy pre-built binaries and certificates
USER 1001:1001
COPY --from=builder /out/lightning /lightning
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
VOLUME [ "/data" ]
WORKDIR /data

# run the bot
ENTRYPOINT ["/lightning"]

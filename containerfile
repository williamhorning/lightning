# build the cli
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o lightning ./cmd/lightning/*.go

# build the final image
FROM scratch

# metadata
LABEL maintainer="William Horning"
LABEL version="0.8.0-alpha.11"
LABEL description="Lightning"
LABEL org.opencontainers.image.title="Lightning"
LABEL org.opencontainers.image.description="extensible chatbot connecting communities"
LABEL org.opencontainers.image.version="0.8.0-alpha.11"
LABEL org.opencontainers.image.source="https://github.com/williamhorning/lightning"
LABEL org.opencontainers.image.licenses="MIT"

# copy stuff over
USER 1001:1001
COPY --from=builder /app/lightning /lightning
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
VOLUME [ "/data" ]
WORKDIR /data

# run the bot
ENTRYPOINT ["/lightning"]
CMD [ "run" ]

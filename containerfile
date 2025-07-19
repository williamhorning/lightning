FROM scratch

# metadata
LABEL maintainer="William Horning"
LABEL version="0.8.0-beta.1"
LABEL description="Lightning"
LABEL org.opencontainers.image.title="Lightning"
LABEL org.opencontainers.image.description="extensible chatbot connecting communities"
LABEL org.opencontainers.image.version="0.8.0-beta.1"
LABEL org.opencontainers.image.source="https://github.com/williamhorning/lightning"
LABEL org.opencontainers.image.licenses="MIT"

# Use build arg to determine architecture
ARG TARGETARCH

# copy pre-built binaries and certificates
USER 1001:1001
COPY ./lightning-${TARGETARCH} /lightning
COPY --from=gcr.io/distroless/static:nonroot /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
VOLUME [ "/data" ]
WORKDIR /data

# run the bot
ENTRYPOINT ["/lightning"]

FROM golang:1.25.1-alpine3.22 AS builder

RUN apk update && apk add --no-cache ca-certificates curl
RUN curl -O https://ssl.com/repo/certs/SSLcom-TLS-Root-2022-ECC.pem && curl -O https://ssl.com/repo/certs/SSL.com-TLS-T-ECC-R2.pem
RUN for cert in *.pem; do cp "$cert" "/usr/local/share/ca-certificates/${cert%.pem}.crt"; done
RUN update-ca-certificates

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -tags goolm -o /out/lightning ./cmd/lightning/main.go

FROM scratch

LABEL maintainer="William Horning"
LABEL version="0.8.0-rc.5"
LABEL description="Lightning"
LABEL org.opencontainers.image.title="Lightning"
LABEL org.opencontainers.image.description="extensible chatbot connecting communities"
LABEL org.opencontainers.image.version="0.8.0-rc.5"
LABEL org.opencontainers.image.source="https://github.com/williamhorning/lightning"
LABEL org.opencontainers.image.licenses="MIT"

USER 1001:1001
COPY --from=builder /out/lightning /lightning
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
VOLUME [ "/data" ]
WORKDIR /data

ENTRYPOINT ["/lightning"]

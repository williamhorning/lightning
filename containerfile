FROM denoland/deno:alpine-2.2.9@sha256:40a7d91338f861e308c3dfe1346f5487d83097cdb898a2e125c05df3cd4841a8

# metadata
LABEL org.opencontainers.image.authors Jersey
LABEL org.opencontainers.image.url https://github.com/williamhorning/bolt
LABEL org.opencontainers.image.source https://github.com/williamhorning/bolt
LABEL org.opencontainers.image.documentation https://williamhorning.eu.org/lightning
LABEL org.opencontainers.image.version 0.8.0
LABEL org.opencontainers.image.licenses MIT
LABEL org.opencontainers.image.title lightning

# install lightning
RUN ["deno", "install", "--global", "--name", "lightning", "--allow-all", "--unstable-temporal", "--unstable-net", "jsr:@jersey/test@0.9.0"]

# run as nobody instead of root
USER nobody

# the volume containing your lightning.toml file
VOLUME [ "/data" ]

# this is the lightning command line
ENTRYPOINT [ "lightning" ]

# run the bot using the user-provided lightning.toml file
CMD [ "run", "/data/lightning.toml" ]

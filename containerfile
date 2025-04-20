FROM denoland/deno:alpine-2.2.11@sha256:c6c801a49a98f295f46176fba6172a1a656decd7dfb086a499fe863f595b922b

# metadata
LABEL org.opencontainers.image.authors Jersey
LABEL org.opencontainers.image.url https://github.com/williamhorning/lightning
LABEL org.opencontainers.image.source https://github.com/williamhorning/lightning
LABEL org.opencontainers.image.documentation https://williamhorning.eu.org/lightning
LABEL org.opencontainers.image.version 0.8.0-alpha.1
LABEL org.opencontainers.image.licenses MIT
LABEL org.opencontainers.image.title lightning

# install lightning
RUN ["deno", "install", "--global", "--name", "lightning", "--allow-all", "--unstable-temporal", "--unstable-net", "jsr:@lightning/test@0.9.0"]

# run as nobody instead of root
USER nobody

# the volume containing your lightning.toml file
VOLUME [ "/data" ]

# this is the lightning command line
ENTRYPOINT [ "lightning" ]

# run the bot using the user-provided lightning.toml file
CMD [ "run", "/data/lightning.toml" ]

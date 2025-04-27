FROM denoland/deno:alpine-2.2.11@sha256:c6c801a49a98f295f46176fba6172a1a656decd7dfb086a499fe863f595b922b

# metadata
LABEL org.opencontainers.image.authors=Jersey
LABEL org.opencontainers.image.url=https://github.com/williamhorning/lightning
LABEL org.opencontainers.image.source=https://github.com/williamhorning/lightning
LABEL org.opencontainers.image.documentation=https://williamhorning.eu.org/lightning
LABEL org.opencontainers.image.version=0.8.0-alpha.2
LABEL org.opencontainers.image.licenses=MIT
LABEL org.opencontainers.image.title=lightning

# make a deno cache directory
RUN ["mkdir", "/deno_dir"]
ENV DENO_DIR=/deno_dir

# install lightning
RUN ["deno", "install", "--global", "--name", "lightning", "--allow-all", "--unstable-temporal", "--unstable-net", "jsr:@lightning/lightning@0.8.0-alpha.2"]
RUN ["chown", "--recursive", "1001:1001", "/deno_dir"]

# run as user instead of root
USER 1001:1001

# the volume containing your lightning.toml file
VOLUME [ "/data" ]

# this is the lightning command line
ENTRYPOINT [ "lightning" ]

# run the bot using the user-provided lightning.toml file
CMD [ "run", "/data/lightning.toml" ]

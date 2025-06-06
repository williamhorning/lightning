FROM denoland/deno:alpine-2.3.5

# make a deno cache directory
RUN ["mkdir", "/deno_dir"]
ENV DENO_DIR=/deno_dir

# install lightning
RUN ["deno", "install", "-gA", "--unstable-temporal", "--unstable-net", "jsr:@lightning/lightning@0.8.0-alpha.5"]
RUN ["chown", "--recursive", "1001:1001", "/deno_dir"]

# run as user instead of root
USER 1001:1001

# the volume containing your lightning.toml file
VOLUME [ "/data" ]
WORKDIR /data

# this is the lightning command line
ENTRYPOINT [ "lightning" ]

# run the bot using the user-provided lightning.toml file
CMD [ "run" ]

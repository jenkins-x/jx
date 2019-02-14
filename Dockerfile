FROM alpine/git

ENTRYPOINT ["jx", "version"]

COPY ./build/linux/jx /usr/bin/jx
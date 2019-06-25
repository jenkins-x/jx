FROM alpine:3.10

RUN apk --no-cache --update add \
    git \
    bash

ENTRYPOINT ["jx", "version"]

COPY ./build/linux/jx /usr/bin/jx

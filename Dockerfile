FROM alpine:3.9.3

RUN apk --update add ca-certificates git

COPY ./build/linux/jx /usr/bin/jx
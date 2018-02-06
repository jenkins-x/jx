# this is used for testing jx inside a cluster in development
FROM jenkinsxio/builder-go:0.0.10

COPY build/jx-linux-amd64 /usr/bin/jx

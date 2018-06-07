FROM scratch

ENTRYPOINT ["/jx", "version"]

COPY build/jx-linux-amd64 /jx


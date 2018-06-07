FROM centos:7

ENTRYPOINT ["jx", "version"]

COPY build/jx-linux-amd64 /usr/bin/jx
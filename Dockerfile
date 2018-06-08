FROM centos:7

ENTRYPOINT ["jx", "version"]

COPY ./build/linux/jx-linux-amd64 /usr/bin/jx

FROM centos:7

ENTRYPOINT ["jx", "version"]

COPY ./build/linux/jx /usr/bin/jx
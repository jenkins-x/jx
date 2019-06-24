FROM centos:9

RUN yum install -y git

ENTRYPOINT ["jx", "version"]

COPY ./build/linux/jx /usr/bin/jx
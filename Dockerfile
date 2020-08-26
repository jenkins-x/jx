FROM gcr.io/jenkinsxio/jx-cli-base:0.0.10

COPY ./build/linux/jx /usr/bin/jx

ENV HOME /home
ENV JX3_HOME /home/.jx3

RUN jx upgrade --mandatory

FROM gcr.io/jenkinsxio-labs-private/jxl-base:0.0.52

COPY ./build/linux/jx /usr/bin/jx

ENV HOME /home
ENV JX3_HOME /home/.jx3

RUN jx upgrade

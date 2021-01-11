FROM gcr.io/jenkinsxio/jx-cli-base-image:0.0.42

COPY ./build/linux/jx /usr/bin/jx

ENV HOME /home
ENV JX3_HOME /home/.jx3

RUN jx upgrade plugins --mandatory

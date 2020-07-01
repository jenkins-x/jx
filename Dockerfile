FROM gcr.io/jenkinsxio-labs-private/jxl-base:0.0.52

COPY ./build/linux/jx /usr/bin/jx

ENV HOME /home
ENV JX_CLI_HOME /home/.jx-cli

RUN jx upgrade

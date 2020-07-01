FROM gcr.io/jenkinsxio-labs-private/jxl-base:0.0.52

COPY ./build/linux/jx /usr/bin/jx

RUN jx upgrade

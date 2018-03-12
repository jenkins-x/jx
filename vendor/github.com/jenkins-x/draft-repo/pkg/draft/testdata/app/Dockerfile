FROM ubuntu:trusty

RUN apt-get update && apt-get install -y \
	nginx \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

# forward request and error logs to docker log collector
RUN ln -sf /dev/stdout /var/log/nginx/access.log
RUN ln -sf /dev/stderr /var/log/nginx/error.log

ENV POWERED_BY Draft

COPY rootfs /

CMD ["/bin/boot"]
EXPOSE 80

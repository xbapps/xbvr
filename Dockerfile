FROM ubuntu:20.04

RUN apt-get update && \
    apt-get install -y --no-install-recommends python3 ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY xbvr /usr/bin/xbvr

EXPOSE 9998-9999
VOLUME /root/.config/

CMD ["/usr/bin/xbvr"]

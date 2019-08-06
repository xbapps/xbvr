FROM ubuntu:19.04
RUN apt update && apt install -y wget ca-certificates

ARG DRONE_TAG

RUN wget -O /tmp/xbvr.tgz "https://github.com/cld9x/xbvr/releases/download/"$DRONE_TAG"/xbvr_"$DRONE_TAG"_Linux_x86_64.tar.gz" && \
    tar xvfz /tmp/xbvr.tgz -C /usr/local/bin/ && \
    rm /tmp/xbvr.tgz

EXPOSE 9999
CMD ["/usr/local/bin/xbvr"]

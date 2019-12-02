FROM ubuntu:19.04 as temp
RUN apt update && apt install -y wget ca-certificates

ARG DRONE_TAG

RUN wget -O /tmp/xbvr.tgz "https://github.com/xbapps/xbvr/releases/download/"$DRONE_TAG"/xbvr_"$DRONE_TAG"_Linux_x86_64.tar.gz" && \
  tar xvfz /tmp/xbvr.tgz -C /usr/local/bin/ && \
  rm /tmp/xbvr.tgz

FROM gcr.io/distroless/base-debian10
COPY --from=temp /usr/local/bin/xbvr /

EXPOSE 9998-9999
VOLUME /root/.config/

ENTRYPOINT ["/xbvr"]

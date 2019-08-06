# Actual image
FROM alpine
RUN apk add --no-cache bash ca-certificates

COPY xbvr /usr/local/bin/xbvr

EXPOSE 9999
ENTRYPOINT ["/usr/local/bin/xbvr"]

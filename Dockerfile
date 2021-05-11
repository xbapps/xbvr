FROM node:12 as build-env

### Install Go ###
ARG TARGETPLATFORM
ENV GO_VERSION=1.13.15 \
    GOPATH=$HOME/go-packages \
    GOROOT=$HOME/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH
RUN if [ "$TARGETPLATFORM" = "linux/arm/v6" ]; then curl -fsSL https://dl.google.com/go/go$GO_VERSION.linux-armv6l.tar.gz | tar -xzv ; elif [ "$TARGETPLATFORM" = "linux/arm/v7" ]; then curl -fsSL https://dl.google.com/go/go$GO_VERSION.linux-armv6l.tar.gz  | tar -xzv ; elif [ "$TARGETPLATFORM" = "linux/arm64" ]; then curl -fsSL https://dl.google.com/go/go$GO_VERSION.linux-arm64.tar.gz | tar -xzv; elif [ "$TARGETPLATFORM" = "linux/amd64" ]; then  curl -fsSL https://dl.google.com/go/go$GO_VERSION.linux-amd64.tar.gz  | tar -xzv;  fi;
RUN GO111MODULE=on go get -u -v \
        github.com/UnnoTed/fileb0x

WORKDIR /app
ADD . /app
RUN cd /app && \
    yarn install && \
    yarn build && \
    go generate && \
    go build -tags='json1' -ldflags '-w' -o xbvr main.go

FROM gcr.io/distroless/base
COPY --from=build-env /app/xbvr /

EXPOSE 9998-9999
VOLUME /root/.config/

ENTRYPOINT ["/xbvr"]

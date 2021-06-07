FROM node:12 as build-env

### Install Go ###
ENV GO_VERSION=1.13.15 \
    GOPATH=$HOME/go-packages \
    GOROOT=$HOME/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH
RUN curl -fsSL https://storage.googleapis.com/golang/go$GO_VERSION.linux-amd64.tar.gz | tar -xzv \
    && GO111MODULE=on go get -u -v \
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

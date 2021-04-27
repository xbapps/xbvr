FROM node:12 as build-env

### Install Go ###
ENV GO_VERSION=1.13.15 \
    GOPATH=$HOME/go-packages \
    GOROOT=$HOME/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH
ENV GO111MODULE=on
RUN curl -fsSL https://storage.googleapis.com/golang/go$GO_VERSION.linux-amd64.tar.gz | tar -xzv \
&& go get -u -v         github.com/acroca/go-symbols \
&&  go get -u -v         github.com/cweill/gotests/... \
&&  go get -u -v         github.com/davidrjenni/reftools/cmd/fillstruct \
&&  go get -u -v         github.com/fatih/gomodifytags \
&&  go get -u -v         github.com/haya14busa/goplay/cmd/goplay \
&&  go get -u -v         github.com/josharian/impl \
&&  go get -u -v         github.com/nsf/gocode \
&&  go get -u -v         github.com/ramya-rao-a/go-outline \
&&  go get -u -v         github.com/rogpeppe/godef \
&&  go get -u -v         github.com/uudashr/gopkgs/cmd/gopkgs \
&&  go get -u -v         github.com/zmb3/gogetdoc \
&&  go get -u -v         golang.org/x/lint/golint \
&&  go get -u -v         golang.org/x/tools/cmd/gorename \
&&  go get -u -v         golang.org/x/tools/cmd/guru \
&&  go get -u -v         sourcegraph.com/sqs/goreturns \
&&  go get -u -v         github.com/UnnoTed/fileb0x

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

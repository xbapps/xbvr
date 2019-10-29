FROM gitpod/workspace-full

ENV HOME=/home/gitpod
WORKDIR $HOME
USER gitpod

ENV GO_VERSION=1.12 \
    GOPATH=$HOME/go-packages \
    GOROOT=$HOME/go
RUN export PATH=$(echo "$PATH" | sed -e 's|:/workspace/go/bin||' -e 's|:/home/gitpod/go/bin||' -e 's|:/home/gitpod/go-packages/bin||')
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

RUN go get -u -v \
    github.com/UnnoTed/fileb0x && \
    # Temp workaround for broken modd deps
    # github.com/cortesi/modd/cmd/modd && \
    git clone https://github.com/cortesi/modd && \
    cd modd && \
    go get mvdan.cc/sh@8aeb0734cd0f && \
    go install ./cmd/modd && \
    sudo rm -rf $GOPATH/src && \
    sudo rm -rf $GOPATH/pkg
# user Go packages
ENV GOPATH=/workspace/go \
    PATH=/workspace/go/bin:$PATH

RUN pip install --no-cache-dir cython && \
    pip install --no-cache-dir flask peewee sqlite-web

USER root

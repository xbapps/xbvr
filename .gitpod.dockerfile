FROM gitpod/workspace-full

ENV HOME=/home/gitpod
WORKDIR $HOME
USER gitpod

ENV GO_VERSION=1.13 \
  GOPATH=$HOME/go-packages \
  GOROOT=$HOME/go
RUN export PATH=$(echo "$PATH" | sed -e 's|:/workspace/go/bin||' -e 's|:/home/gitpod/go/bin||' -e 's|:/home/gitpod/go-packages/bin||')
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

RUN GO111MODULE=on go get -u -v \
  github.com/UnnoTed/fileb0x \
  github.com/cosmtrek/air && \
  sudo rm -rf $GOPATH/src && \
  sudo rm -rf $GOPATH/pkg
# user Go packages
ENV GOPATH=/workspace/go \
  PATH=/workspace/go/bin:$PATH

RUN pip3 install --no-cache-dir cython && \
  pip3 install --no-cache-dir flask peewee sqlite-web

USER root

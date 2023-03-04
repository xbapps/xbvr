FROM gitpod/workspace-full

ENV HOME=/home/gitpod
WORKDIR $HOME
USER gitpod

ENV GO_VERSION=1.19.6 \
  GOPATH=$HOME/go-packages \
  GOROOT=$HOME/go
RUN export PATH=$(echo "$PATH" | sed -e 's|:/workspace/go/bin||' -e 's|:/home/gitpod/go/bin||' -e 's|:/home/gitpod/go-packages/bin||')
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

RUN go install -v \
  github.com/cosmtrek/air@latest && \
  sudo rm -rf $GOPATH/src && \
  sudo rm -rf $GOPATH/pkg
# user Go packages
ENV GOPATH=/workspace/go \
  PATH=/workspace/go/bin:$PATH

RUN pip3 install --no-cache-dir cython && \
  pip3 install --no-cache-dir flask peewee sqlite-web

USER root

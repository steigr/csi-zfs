FROM golang:1.11.4 AS builder

ARG CSI_RELEASE=v1.0.0
ENV CSI_RELEASE=$CSI_RELEASE \
    CSI_PKG=github.com/container-storage-interface/spec \
    CSI_PROTOBUF_GO=lib/go/csi/csi.pb.go

RUN  curl -L https://$CSI_PKG/raw/$CSI_RELEASE/$CSI_PROTOBUF_GO \
     | install -D /dev/stdin $GOPATH/src/$CSI_PKG/$CSI_PROTOBUF_GO \
 &&  printf 'module %s' "$CSI_PKG" | install -D /dev/stdin $GOPATH/src/$CSI_PKG/go.mod

ARG PKG=github.com/steigr/csi-zfs
ENV PKG=$PKG \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on
WORKDIR /go/src/$PKG
COPY go.mod go.mod
COPY go.sum go.sum
RUN  go mod download
COPY cmd/csi-zfs cmd/csi-zfs
COPY pkg pkg
RUN  go get -v $PKG/cmd/csi-zfs

FROM alpine:3.9 AS csi-zfs
COPY --from=builder /go/bin/csi-zfs /bin/csi-zfs
ENTRYPOINT ["csi-zfs"]

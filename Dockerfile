FROM golang:1.11

COPY . /go/src/github.com/mittwald/kube-pod-director
WORKDIR /go/src/github.com/mittwald/kube-pod-director
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kube-pod-director .

FROM scratch
MAINTAINER Martin Helmich <m.helmich@mittwald.de>

COPY --from=0 /go/src/github.com/mittwald/kube-pod-director/kube-pod-director /kube-pod-director

ENTRYPOINT ["/kube-pod-director"]

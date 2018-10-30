FROM golang:1.11

COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kube-pod-director .

FROM scratch
MAINTAINER Martin Helmich <m.helmich@mittwald.de>

COPY --from=0 /app/kube-pod-director /kube-pod-director

ENTRYPOINT ["/kube-pod-director"]

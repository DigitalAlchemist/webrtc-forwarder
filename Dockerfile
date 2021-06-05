FROM golang:1.16 as builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o webrtc-forwarder *.go

FROM busybox:latest
COPY --from=builder /build/webrtc-forwarder /bin


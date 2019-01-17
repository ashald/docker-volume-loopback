# build
FROM golang:1.11-alpine as builder
RUN apk --no-cache add make
ADD ./ /go/src/github.com/ashald/docker-volume-loopback/
WORKDIR /go/src/github.com/ashald/docker-volume-loopback
RUN make build && \
    mv ./docker-volume-loopback /usr/bin/


# package
FROM alpine
RUN apk --no-cache add e2fsprogs
COPY --from=builder /usr/bin/docker-volume-loopback /docker-volume-loopback
CMD [ "/docker-volume-loopback" ]

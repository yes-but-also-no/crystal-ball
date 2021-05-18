FROM golang:1.16.3-alpine as builder

WORKDIR /go/src/app
COPY . .

RUN apk add git gcc g++ linux-headers

RUN go get -d ./...
RUN go install -v -trimpath ./...

FROM alpine:3.13.5

COPY --from=builder /go/bin/* /bin/
RUN mkdir -p /orakuru/etc
WORKDIR /orakuru

CMD ["/bin/crystal-ball"]

FROM golang:1.8.3-alpine as builder

RUN apk add --no-cache \
  ca-certificates \
  && update-ca-certificates

ENV GOPATH /go

WORKDIR /go/src/github.com/UKHomeOffice/smilodon
COPY . .

RUN mkdir -p /go/bin
RUN go build -o /go/bin/smilodon

FROM alpine

RUN apk update \
  && apk add --no-cache \
  ca-certificates \
  && update-ca-certificates

COPY --from=builder /go/bin/smilodon /go/bin/smilodon

CMD "./go/bin/smilodon"

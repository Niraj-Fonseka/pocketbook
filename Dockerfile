FROM golang:1.17.5-stretch as builder

COPY . /pocketbook
WORKDIR /pocketbook

ENV GO111MODULE=on

RUN CGO_ENABLED=0 GOOS=linux go build -o pocketbook-app 


FROM alpine:latest

RUN apk add --no-cache tzdata

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /root/

COPY --from=builder /pocketbook .

CMD ["./pocketbook-app"]
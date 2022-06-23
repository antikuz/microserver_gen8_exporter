FROM golang:1.18.3-alpine3.16 AS builder

WORKDIR /build

COPY . ./
RUN go mod download \
 && go build -o microserver_gen8_exporter


FROM alpine:3.16

WORKDIR /app

COPY --from=builder /build/microserver_gen8_exporter ./microserver_gen8_exporter

EXPOSE 8080

ENTRYPOINT ["/app/microserver_gen8_exporter"]
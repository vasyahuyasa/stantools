FROM golang:1.13.3-alpine3.10 AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -o stanrate/stanrate ./stanrate

FROM alpine:3.10
WORKDIR /app
COPY --from=builder /build/stanrate/stanrate /app/stanrate
CMD ["/bin/sh", "-c", "--", "while true; do sleep 5; done;"]

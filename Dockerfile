FROM golang:1.13
WORKDIR /root/src/
COPY . /root/src/
RUN CGO_ENABLED=0 go build -v .

FROM alpine:latest
LABEL maintainer="maksim77ster@gmail.com"
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /root/src/tinkoff_exporter .
CMD ["./tinkoff_exporter"]

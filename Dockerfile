FROM golang:1.9-alpine3.6
COPY . /go/src/github.com/DavidHuie/vespyr
RUN go install github.com/DavidHuie/vespyr/cmd/vespyr

FROM alpine:3.6
WORKDIR /bin
COPY --from=0 /go/bin/vespyr .
RUN apk --update add ca-certificates
CMD ["./app"]
ENTRYPOINT ["/bin/vespyr"]
CMD [""]
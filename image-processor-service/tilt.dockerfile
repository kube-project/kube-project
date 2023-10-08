FROM alpine
WORKDIR /
COPY ./bin/processor /processor

ENTRYPOINT ["/processor"]

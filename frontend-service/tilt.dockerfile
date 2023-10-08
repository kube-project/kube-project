FROM alpine
WORKDIR /
COPY ./bin/frontend /frontend
COPY ./index.html index.html

EXPOSE 8081

ENTRYPOINT ["/frontend"]

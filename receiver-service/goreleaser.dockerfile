FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY receiver-service /app/receiver
USER 65532:65532

EXPOSE 8000

ENTRYPOINT [ "/app/receiver" ]
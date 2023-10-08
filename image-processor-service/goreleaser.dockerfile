FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY image-processor-service /app/processor
USER 65532:65532



ENTRYPOINT [ "/app/processor" ]
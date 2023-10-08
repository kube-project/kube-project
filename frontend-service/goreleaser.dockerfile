FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY frontend-service /app/frontend
USER 65532:65532

EXPOSE 8081

ENTRYPOINT [ "/app/frontend" ]
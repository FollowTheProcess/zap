FROM gcr.io/distroless/static
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/zap /usr/local/bin/zap
ENTRYPOINT [ "/usr/local/bin/zap" ]

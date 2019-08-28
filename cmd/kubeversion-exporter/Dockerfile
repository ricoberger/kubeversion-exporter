FROM alpine:latest

ARG REVISION
ARG VERSION

LABEL maintainer="Rico Berger"
LABEL git.ref=$REVISION
LABEL git.version=$VERSION
LABEL git.url="https://github.com/ricoberger/kubeversion-exporter"

RUN apk add --no-cache --update curl ca-certificates
HEALTHCHECK --interval=10s --timeout=3s --retries=3 CMD curl --fail http://localhost:9637/ || exit 1

RUN addgroup -g 1000 kubeversion-exporter && \
    adduser -D -u 1000 -G kubeversion-exporter kubeversion-exporter
USER kubeversion-exporter

COPY ./bin/kubeversion-exporter-linux-amd64  /bin/kubeversion-exporter
EXPOSE 8080

ENTRYPOINT  [ "/bin/kubeversion-exporter" ]

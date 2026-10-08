FROM alpine:3.24

LABEL    org.opencontainers.image.source="https://github.com/cz-lucas/evcc-prometheus"
LABEL    org.opencontainers.image.licenses="Apache-2.0"
LABEL    org.opencontainers.image.title="evcc-prometheus"
LABEL    org.opencontainers.image.description="Prometheus exporter for evcc"



ARG TARGETARCH

RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates \
    && addgroup -S exporter \
    && adduser -S -G exporter exporter

COPY bin/exporter-${TARGETARCH} /usr/local/bin/evcc-prometheus
RUN chmod 0555 /usr/local/bin/evcc-prometheus

ENV PROMETHEUS_ADDR=:9070 \
    HEALTHCHECK_URL=http://localhost:9070/health

USER exporter:exporter

EXPOSE 9070

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/bin/sh", "-c", "wget -q -O /dev/null \"$HEALTHCHECK_URL\" || exit 1"]

ENTRYPOINT ["/usr/local/bin/evcc-prometheus"]
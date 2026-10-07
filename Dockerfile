FROM alpine:latest

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
FROM ubuntu:18.04

COPY azure-cns azure-cns

COPY azure-vnet-telemetry azure-vnet-telemetry

RUN chmod +x azure-cns;

EXPOSE 10090

ENTRYPOINT ["/azure-cns", "-c", "tcp://localhost:10090", "-stopcnm=true"]

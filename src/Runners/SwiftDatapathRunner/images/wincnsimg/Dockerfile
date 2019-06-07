FROM mcr.microsoft.com/windows/servercore:ltsc2019

COPY azure-cns.exe azure-cns.exe

COPY azure-vnet-telemetry.exe azure-vnet-telemetry.exe

EXPOSE 10090

ENTRYPOINT ["azure-cns.exe", "-c", "tcp://0.0.0.0:10090", "-stopcnm=true"]


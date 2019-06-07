FROM mcr.microsoft.com/windows/servercore:ltsc2019

COPY azure-dnc.exe azure-dnc.exe

COPY config.json config.json

EXPOSE 9000

ENTRYPOINT ["azure-dnc.exe"]


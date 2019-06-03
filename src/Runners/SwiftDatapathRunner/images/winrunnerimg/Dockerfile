FROM mcr.microsoft.com/windows/servercore:ltsc2019

COPY swiftrunner.exe swiftrunner.exe

COPY container0.yaml container0.yaml

COPY container1.yaml container1.yaml

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/windows/amd64/kubectl.exe; \
    mv .\kubectl.exe C:\Windows\System32\systemkubectl.exe

EXPOSE 9009

ENTRYPOINT ["swiftrunner.exe"]


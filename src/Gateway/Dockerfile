#   To build Gateway docker image, follow below steps
#
#   a. Change to src directory on your machine in networking aquarius repo
#      cd Networking/Aquarius/src
#
#   b. Run docker build as below
#      docker build -t dnc-gateway:version -f Gateway/Dockerfile --no-cache .
#
#   ex1.  docker build -t acndev.azurecr.io/dnc-gateway:latest -f Gateway/Dockerfile --no-cache .
#   ex2.  docker build -t acnprod.azurecr.io/dnc-gateway:latest -f Gateway/Dockerfile --no-cache .

FROM mcr.microsoft.com/dotnet/core/sdk:3.1 AS builder

# Create directory for source
WORKDIR /usr/local/Networking-Aquarius/src

# Copy Gateway code and its dependencies
COPY ./Gateway/Server/ ./Gateway/Server
COPY ./Common/CoreLib ./Common/CoreLib

# Create out directory for binaries
RUN mkdir -p /usr/local/Gateway

WORKDIR /usr/local/Networking-Aquarius/src/Gateway/Server/src
RUN dotnet publish -c Release -r linux-x64 -o /usr/local/Gateway --self-contained true

# Build runtime image
FROM mcr.microsoft.com/dotnet/core/aspnet:3.1
WORKDIR /app
COPY --from=builder /usr/local/Gateway .

EXPOSE 5001

ENTRYPOINT ["/app/Gateway"]



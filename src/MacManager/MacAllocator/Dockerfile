#   To build MacAllocator docker image, follow below steps
#
#   a. Create a PAT (personal access token) for azureacnfeed feed in your devops profile. Copy out the value of PAT and use it in step c)
#
#   b. Change to src directory on your machine in networking aquarius repo
#      cd Networking/Aquarius/src
#
#   c. Run docker build as below
#      docker build -t acnprod.azurecr.io/macallocator:version -f MacManager/MacAllocator/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=<PAT> --build-arg USERNAME=<username>.
#
#   ex1. docker build -t acnprod.azurecr.io/macallocator:latest -f MacManager/MacAllocator/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=abcdef --build-arg USERNAME=ghij .
#   ex2. docker build -t acndev.azurecr.io/macallocator:latest -f MacManager/MacAllocator/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=abcdefghijk --build-arg USERNAME=ghij .
#
#   For local/dev testing, set ASPNETCORE_ENVIRONMENT to "Development" and also set AzureServicesAuthConnectionString retrieved from keyvault. you can set these values in --build arg.
#   ex3. docker build -t macallocator:latest -f MacManager/MacAllocator/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=abcdef --build-arg USERNAME=ghij --build-arg ASPNETCORE_ENVIRONMENT=Development --build-arg="AzureServicesAuthConnectionString=RunAs=App;AppId=xxx;TenantId=xxx;AppKey=xxx" .

FROM mcr.microsoft.com/dotnet/sdk:5.0 AS builder
RUN apt -y update && apt -y upgrade
RUN curl -L https://raw.githubusercontent.com/Microsoft/artifacts-credprovider/master/helpers/installcredprovider.sh  | sh

ARG FEED_ACCESSTOKEN
ARG USERNAME
ENV VSS_NUGET_EXTERNAL_FEED_ENDPOINTS \
    "{\"endpointCredentials\": [{\"endpoint\":\"https://msazure.pkgs.visualstudio.com/one/_packaging/azureacnfeed/nuget/v3/index.json\", \"username\":\"${USERNAME}\", \"password\":\"${FEED_ACCESSTOKEN}\"}]}"
# Create directory for source
WORKDIR /usr/local/Networking-Aquarius/src

# Copy MacAllocator code and its dependencies
COPY ./MacManager/common/ ./MacManager/common
COPY ./MacManager/MacAllocator/Server/ ./MacManager/MacAllocator/Server
COPY ./Common/CoreLib ./Common/CoreLib

# Create out directory for binaries
RUN mkdir -p /usr/local/MacAllocator

WORKDIR /usr/local/Networking-Aquarius/src/MacManager/MacAllocator/Server

RUN dotnet publish -c Release -r linux-musl-x64 -o /usr/local/MacAllocator --self-contained true

# Build runtime image
FROM mcr.microsoft.com/dotnet/runtime:5.0-alpine-amd64
ARG ASPNETCORE_ENVIRONMENT="Production"
ENV ASPNETCORE_ENVIRONMENT $ASPNETCORE_ENVIRONMENT
ARG AzureServicesAuthConnectionString=""
ENV AzureServicesAuthConnectionString $AzureServicesAuthConnectionString 
WORKDIR /app
COPY --from=builder /usr/local/MacAllocator .

EXPOSE 5012

ENTRYPOINT ["/app/MacAllocator"]



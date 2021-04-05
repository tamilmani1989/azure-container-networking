#   To build DncRP docker image, follow below steps
#
#   a. Create a PAT (personal access token) for msblox feed in your devops profile. Copy out the value of PAT and use it in step c)
#
#   b. Change to src directory on your machine in networking aquarius repo
#      cd Networking/Aquarius/src
#
#   c. Run docker build as below
#      docker build -t dncrp:version -f DncRP/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=<PAT> --build-arg USER_NAME=<User alias> .
#
#   ex1. docker build -t acnprod.azurecr.io/dncrp:latest -f DncRP/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=abcdef --build-arg USER_NAME=akdkd .
#   ex2. docker build -t acndev.azurecr.io/dncrp:latest -f DncRP/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=abcdefghijk --build-arg USER_NAME=akdkd .
#
#   For local/dev testing, set ASPNETCORE_ENVIRONMENT to "Development" and also set AzureServicesAuthConnectionString retrieved from keyvault. you can set these values in --build arg.
#   ex3. docker build -t dncrp:latest -f DncRP/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=abcdef --build-arg ASPNETCORE_ENVIRONMENT=Development --build-arg="AzureServicesAuthConnectionString=RunAs=App;AppId=xxx;TenantId=xxx;AppKey=xxx" .

FROM mcr.microsoft.com/dotnet/sdk:5.0 AS builder
RUN apt -y update && apt -y upgrade
RUN curl -L https://raw.githubusercontent.com/Microsoft/artifacts-credprovider/master/helpers/installcredprovider.sh  | sh

ARG FEED_ACCESSTOKEN
ARG USER_NAME
ENV VSS_NUGET_EXTERNAL_FEED_ENDPOINTS \
    "{\"endpointCredentials\": [{\"endpoint\":\"https://msazure.pkgs.visualstudio.com/one/_packaging/networking-aquarius-Consumption/nuget/v3/index.json\", \"username\":\"${USER_NAME}\", \"password\":\"${FEED_ACCESSTOKEN}\"}]}"
# Create directory for source
WORKDIR /usr/local/Networking-Aquarius/src

# Copy DncRP code and its dependencies
COPY ./DncRP/Server/ ./DncRP/Server
COPY ./Common/CoreLib ./Common/CoreLib

# Create out directory for binaries
RUN mkdir -p /usr/local/DncRP

WORKDIR /usr/local/Networking-Aquarius/src/DncRP/Server

RUN dotnet publish -c Release -r linux-musl-x64 -o /usr/local/DncRP --self-contained true

# Build runtime image
FROM mcr.microsoft.com/dotnet/runtime:5.0-alpine-amd64
ARG ASPNETCORE_ENVIRONMENT="Production"
ENV ASPNETCORE_ENVIRONMENT $ASPNETCORE_ENVIRONMENT
ARG AzureServicesAuthConnectionString=""
ENV AzureServicesAuthConnectionString $AzureServicesAuthConnectionString 
WORKDIR /app
COPY --from=builder /usr/local/DncRP .

EXPOSE 5002

ENTRYPOINT ["/app/DncRP"]



#   To build DncRP docker image, follow below steps
#
#   a. Create a PAT (personal access token) for msblox feed in your devops profile. Copy out the value of PAT and use it in step c)
#
#   b. Change to src directory on your machine in networking aquarius repo
#      cd Networking/Aquarius/src
#
#   c. Run docker build as below
#      docker build -t dncrp:version -f DncRP/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=<PAT> .
#
#   ex1. docker build -t acnprod.azurecr.io/dncrp:latest -f DncRP/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=abcdefghijk .
#   ex2. docker build -t acndev.azurecr.io/dncrp:latest -f DncRP/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN=abcdefghijk .

FROM mcr.microsoft.com/dotnet/core/sdk:3.1 AS builder

RUN curl -L https://raw.githubusercontent.com/Microsoft/artifacts-credprovider/master/helpers/installcredprovider.sh  | sh

ARG FEED_ACCESSTOKEN
ENV VSS_NUGET_EXTERNAL_FEED_ENDPOINTS \
    "{\"endpointCredentials\": [{\"endpoint\":\"https://msblox.pkgs.visualstudio.com/_packaging/AzureGenevaMonitoring/nuget/v3/index.json\", \"username\":\"vik\", \"password\":\"${FEED_ACCESSTOKEN}\"}]}"

# Create directory for source
WORKDIR /usr/local/Networking-Aquarius/src

# Copy DncRP code and its dependencies
COPY ./DncRP/Server/ ./DncRP/Server
COPY ./Common/CoreLib ./Common/CoreLib

# Create out directory for binaries
RUN mkdir -p /usr/local/DncRP

WORKDIR /usr/local/Networking-Aquarius/src/DncRP/Server
RUN dotnet publish -c Release -r linux-x64 -o /usr/local/DncRP --self-contained true

# Build runtime image
FROM mcr.microsoft.com/dotnet/core/aspnet:3.1
WORKDIR /app
COPY --from=builder /usr/local/DncRP .

EXPOSE 5002

ENTRYPOINT ["/app/DncRP"]



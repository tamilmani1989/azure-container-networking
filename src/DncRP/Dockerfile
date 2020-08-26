#   To build DncRP docker image, follow below steps
#
#   a. Change to src directory on your machine in networking aquarius repo
#      cd Networking/Aquarius/src
#
#   b. Run docker build as below
#      docker build -t dncrp:version -f DncRP/Dockerfile --no-cache .

FROM mcr.microsoft.com/dotnet/core/sdk:3.1 AS builder

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



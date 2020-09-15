# Build requestcontroller
FROM golang:1.14-alpine AS builder

# Get deps
RUN apk --update add ca-certificates curl unzip

# Setup non-root user
ENV USER=appuser
ENV UID=1000
RUN adduser \
	--disabled-password \
	--gecos "" \
	--home "/home/appuser" \
	--shell "/sbin/nologin" \
	--no-create-home \
	--uid "${UID}" \
	"${USER}"

WORKDIR /usr/local/src/dnc

# Copy the source, we don't have to be picky here since this is an intermediate container
COPY . .

# Build
RUN cd cmd/requestcontroller && \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /usr/local/bin/requestcontroller .

# Get kubelogin binary
WORKDIR /usr/local/kubelogin
RUN curl -sSLO https://github.com/Azure/kubelogin/releases/download/v0.0.6/kubelogin-linux-amd64.zip && \
	unzip kubelogin-linux-amd64.zip

# Copy into final image
FROM scratch
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder --chown=appuser:appuser /usr/local/bin/requestcontroller \
	/usr/local/bin/requestcontroller
COPY --from=builder --chown=appuser:appuser /usr/local/kubelogin/bin/linux_amd64/kubelogin /usr/local/bin/kubelogin
COPY --from=builder --chown=appuser:appuser /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER appuser:appuser

ENTRYPOINT [ "/usr/local/bin/requestcontroller" ]

FROM alpine:3.8

ENV ENV="/etc/profile"

RUN apk update; \
    apk add --no-cache bash; \
    apk add --no-cache curl; \
    apk add --no-cache ca-certificates
    
RUN echo "echo 'nameserver 168.63.129.16' > /etc/resolv.conf" >> /etc/profile

CMD ["/bin/sh", "-c", "tail -f /dev/null"]

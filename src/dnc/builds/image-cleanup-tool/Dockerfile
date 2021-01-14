FROM alpine:latest

RUN apk update && apk add --no-cache docker-cli

RUN echo "#!/bin/sh" > /usr/sbin/image-cleanup.sh && echo 'while [ true ]; do \
docker image prune -a --force; \
sleep 1h; \
done' >> /usr/sbin/image-cleanup.sh

RUN chmod +x /usr/sbin/image-cleanup.sh

ENTRYPOINT ["/usr/sbin/image-cleanup.sh"]
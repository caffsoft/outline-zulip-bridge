# Build a tiny docker image
FROM alpine:3.21

RUN mkdir /app

COPY ./bin/outline-zulip-bridge /app

CMD [ "/app/outline-zulip-bridge" ]

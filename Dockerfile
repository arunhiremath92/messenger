FROM golang:1.26-alpine

RUN apk add --no-cache make git

WORKDIR /app

COPY . /app/

RUN make all

CMD [ "/app/bin/mgclient" ]
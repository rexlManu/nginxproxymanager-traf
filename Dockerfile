# syntax=docker/dockerfile:1

FROM golang:1.17-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /npm-traf

ENV INFLUXDB_BUCKET=bucket
ENV INFLUXDB_ORG=org
ENV INFLUXDB_TOKEN=token
ENV INFLUXDB_URL=url

CMD [ "/npm-traf" ]
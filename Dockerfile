FROM golang:alpine
WORKDIR /go/src/github.com/pippozq/pushgateway

ENV  S___LOG_LEVEL 5
ENV  S___POOLSIZE 200
ENV  S___REDISAGENT_POOL_IDLETIMEOUT 0
ENV  S___REDISAGENT_REDISDB 1
ENV  S___REDISAGENT_REDISEXPIRETIME 70
ENV  S___REDISAGENT_REDISHOST 172.16.21.59
ENV  S___REDISAGENT_REDISPASSWORD redis
ENV  S___REDISAGENT_REDISPORT 36379

RUN go build
EXPOSE 80
ENTRYPOINT ["/go/src/github.com/pippozq/pushgateway/pushgateway"]
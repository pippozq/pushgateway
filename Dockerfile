FROM golang:1.11 as builder
WORKDIR /go/src/github.com/pippozq/pushgateway
COPY . .

RUN CGO_ENABLED=0 GO111MODULE=on go build

FROM golang:1.11-alpine

ENV  GOENV DEV
ENV  S___Log_Level debug
ENV  S___PoolSize 200
ENV  S___RedisAgent_KeyCount 100
ENV  S___RedisAgent_PipelineWaitTime 3
ENV  S___RedisAgent_RedisDb 1
ENV  S___RedisAgent_RedisExpireTime 70
ENV  S___RedisAgent_RedisHost 127.0.0.1
ENV  S___RedisAgent_RedisPassword redis
ENV  S___RedisAgent_RedisPort 6379
ENV  S___Server_Port 80

EXPOSE 80

COPY --from=builder /go/src/github.com/pippozq/pushgateway/pushgateway ./pushgateway
COPY --from=builder /go/src/github.com/pippozq/pushgateway/config ./config

ENTRYPOINT ["./pushgateway"]
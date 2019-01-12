# Pushgateway
## Use for
1. Receive metrics from http and cache them in redis
2. Provide metrics as text for Prometheus
## Difference from [official pushgateway](https://github.com/prometheus/pushgateway)
1. Use redis as cache, instead of memory, can be used as cluster
2. Every metric can be set different expire time
3. With no web ui
4. Use json format data
## Docker
1. Env Config

```
S___Log_Level debug                  // log level
S___RedisAgent_PoolSize 200          //  redis pool
S___RedisAgent_KeyCount 100
S___RedisAgent_PipelineWaitTime 3    
S___RedisAgent_RedisDb 1
S___RedisAgent_RedisExpireTime 70
S___RedisAgent_RedisHost 127.0.0.1
S___RedisAgent_RedisPassword redis
S___RedisAgent_RedisPort 6379
S___Server_Port 80
```

2. Build
```
docker build -t pushgateway:1 .
```
3. Provide k8s-srv.yml

## How to use?
1. Json Format

```
{
  "expire_time": 60,                // metric expire time，default 1800s
  "id": "192.168.3.4",              // primary id,such like ip or something unique
  "job_name": "job_name",           // your job name
  "metrics": [
    {
      "metric_name": "cpu_usage",   // metric name
      "metric_value": 71,           // value  float64
      "labels": {
        "os": "linux"
      }
    }
  ]
}
```
2. Push data with POST Method
```
127.0.0.1:8000/pushgateway/v0/metrics
```

3. Prometheus Config

```
- job_name: pushgateway
  scrape_interval: 1m   // as you wish
  scrape_timeout: 50s   // as you wish too
  metrics_path: /pushgateway/v0/metrics
  scheme: http
  static_configs:
  - targets:
    - your ip
```


# License
MIT License
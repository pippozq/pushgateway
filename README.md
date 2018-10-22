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
S___SERVER_PORT=8000                // http service port
S___POOLSIZE=200                    // goroutine pool size
S___REDISAGENT_MAXACTIVE=50         // redis conf
S___REDISAGENT_MAXIDLE=10
S___REDISAGENT_REDISDB=1
S___REDISAGENT_REDISEXPIRETIME=1800
S___REDISAGENT_REDISHOST=192.16.3.40
S___REDISAGENT_REDISPASSWORD=redis
S___REDISAGENT_REDISPORT=6379
```

2. Build
```
docker build -t pushgateway:1 .
```
3. Provide docker-compose.yml

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
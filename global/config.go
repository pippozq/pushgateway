package global

import (
	"github.com/pippozq/pushgateway/modules/redis"
	"github.com/go-courier/httptransport"
	"github.com/sirupsen/logrus"
)

func init() {
	Config.RedisAgent.Pool = Config.RedisAgent.InitPool()
	Config.Server.SetDefaults()
}


var Config = struct {
	Log    *logrus.Logger
	Server  httptransport.HttpTransport
	RedisAgent      *redis.Agent
	PoolSize   int `conf:"env"`
}{
	Log: &logrus.Logger{
		Level: logrus.DebugLevel,
	},
	Server: httptransport.HttpTransport{
		Port:     8000,
	},

	RedisAgent:&redis.Agent{
		RedisHost:       "172.16.21.59",
		RedisPort:       "36379",
		RedisPassword:   "redis",
		RedisDb:         "1",
		RedisExpireTime: "70",
	},
	PoolSize: 200,
}

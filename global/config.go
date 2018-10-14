package global

import (
	"github.com/go-courier/envconf"
	"github.com/go-courier/httptransport"
	"github.com/pippozq/pushgateway/modules/redis"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

func init() {
	Config.RedisAgent.Pool = Config.RedisAgent.InitPool()
	Config.Server.SetDefaults()

	// inject env
	envVars := envconf.NewEnvVars("S")
	envVars = envconf.EnvVarsFromEnviron("S", os.Environ())
	envconf.NewDotEnvDecoder(envVars).Decode(&Config)
	data, _ := envconf.NewDotEnvEncoder(envVars).Encode(&Config)

	for _, env := range strings.Split(string(data), "\n") {
		if env != "" {
			logrus.Println("ENV ", env)
		}
	}

}

var Config = struct {
	Log        *logrus.Logger
	Server     httptransport.HttpTransport
	RedisAgent *redis.Agent
	PoolSize   int `conf:"env"`
}{
	Log: &logrus.Logger{
		Level: logrus.DebugLevel,
	},
	Server: httptransport.HttpTransport{
		Port: 8000,
	},

	RedisAgent: &redis.Agent{
		RedisHost:       "172.16.21.59",
		RedisPort:       "36379",
		RedisPassword:   "redis",
		RedisDb:         "1",
		RedisExpireTime: "70",
	},
	PoolSize: 200,
}

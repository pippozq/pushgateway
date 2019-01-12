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
	Config.Server.SetDefaults()

	// inject env
	envVars := envconf.NewEnvVars("S")
	envVars = envconf.EnvVarsFromEnviron("S", os.Environ())
	if err := envconf.NewDotEnvDecoder(envVars).Decode(&Config); err != nil {
		logrus.Error(err)
	}

	data, _ := envconf.NewDotEnvEncoder(envVars).Encode(&Config)

	for _, env := range strings.Split(string(data), "\n") {
		if env != "" {
			logrus.Println("ENV ", env)
		}
	}

	Config.RedisAgent.Pool = Config.RedisAgent.InitPool()
}

var Config = struct {
	Log        *logrus.Logger
	Server     httptransport.HttpTransport
	RedisAgent *redis.Agent
}{

	Log : logrus.New(),
	RedisAgent: &redis.Agent{
		RedisHost:        "127.0.0.1",
		RedisPort:        6379,
		RedisPassword:    "",
		RedisDb:          1,
		RedisExpireTime:  70,
		PoolSize:         100,
		KeyCount:         100,
		PipelineWaitTime: 3,
	},
}

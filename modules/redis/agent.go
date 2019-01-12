package redis

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/pippozq/pushgateway/constants/errors"
	"github.com/sirupsen/logrus"
	"time"
)

type Agent struct {
	Pool             *redis.Client
	RedisHost        string `conf:"env"`
	RedisPort        int    `conf:"env"`
	RedisPassword    string `conf:"env"`
	RedisDb          int    `conf:"env"`
	RedisExpireTime  int    `conf:"env"`
	PoolSize         int    `conf:"env"`
	KeyCount         int    `conf:"env"` // 单次获取的最大key数量
	PipelineWaitTime int    `conf:"env"` // pipeline 等待时间 单位秒
}

func (agent *Agent) InitPool() *redis.Client {

	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", agent.RedisHost, agent.RedisPort),
		Password: agent.RedisPassword, // no password set
		DB:       agent.RedisDb,       // use default DB
		PoolSize: agent.PoolSize,
	})
}

//  1  exist  0 not exist
func (agent *Agent) CheckKeyExist(key string) (exist int64, err error) {
	return agent.Pool.Exists(key).Result()
}

func (agent *Agent) GetKeyList(key string) (keys []string, err error) {
	return agent.Pool.Keys(key).Result()
}

func (agent *Agent) Set(key string, value []byte, expire int) (err error) {

	if expire <= 0 {
		err = agent.Pool.Set(key, value, time.Duration(0)).Err()
	} else {
		err = agent.Pool.Set(key, value, time.Duration(expire)*time.Second).Err()
	}

	if err != nil {
		logrus.Errorf("redis Set failed:", err)
		return err
	}
	return nil
}

func (agent *Agent) Get(key string) (value []byte, err error) {

	value, err = agent.Pool.Get(key).Bytes()
	if err == redis.Nil {
		return nil, errors.MetricNotFound
	} else if err != nil {
		return nil, err
	}
	return
}

func (agent *Agent) MGet(key ...string) (values []interface{}, err error) {
	values, err = agent.Pool.MGet(key...).Result()
	if err == redis.Nil {
		return nil, errors.MetricNotFound
	} else if err != nil {
		return nil, err
	}
	return
}

func (agent *Agent) Del(key string) (err error) {
	return agent.Pool.Del(key).Err()
}

func (agent *Agent) Publish(channel string, message string) (err error) {
	return agent.Pool.Publish(channel, message).Err()
}

func (agent *Agent) Subscribe(channel string) *redis.PubSub {
	return agent.Pool.Subscribe(channel)
}

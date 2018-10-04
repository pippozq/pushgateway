package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/sirupsen/logrus"
)



type Agent struct {
	Pool redis.Pool
	// redis
	RedisHost       string `conf:"env"`
	RedisPort       string `conf:"env"`
	RedisPassword   string `conf:"env"`
	RedisDb         string `conf:"env"`
	RedisExpireTime string `conf:"env"`
}

func (agent *Agent)InitPool() redis.Pool {
	return redis.Pool{
		Wait:      true,
		MaxIdle:   120,
		MaxActive: 100,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", agent.RedisHost, agent.RedisPort))
			if err != nil {
				logrus.Errorf("Connect to redis error", err)
				return c, err
			}
			if agent.RedisPassword != "" {
				if _, err := c.Do("AUTH", agent.RedisPassword); err != nil {
					c.Close()
					panic(err)
				}
			}
			return c, err
		},
	}
}

func (agent *Agent) selectDB(db string) (conn redis.Conn, err error) {
	c := agent.Pool.Get()
	if c.Err() != nil {
		logrus.Error(c.Err())
		return nil, err
	}
	_, err = c.Do("SELECT", db)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return c, nil
}

func (agent *Agent) GetKeyList(key string) (keys []string, err error) {

	c, err := agent.selectDB(agent.RedisDb)
	defer c.Close()
	if err != nil {
		return nil, err
	}

	keys, err = redis.Strings(c.Do("KEYS", key))
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (agent *Agent) Set(key string, value []byte, expire string) (err error) {
	c, err := agent.selectDB(agent.RedisDb)
	defer c.Close()
	if err != nil {
		return err
	}

	_, err = c.Do("SET", key, value, "EX", expire)
	if err != nil {
		logrus.Errorf("redis Set failed:", err)
		return err
	}
	return nil
}

func (agent *Agent) Get(key string) (valueBytes []byte, err error) {
	c, err := agent.selectDB(agent.RedisDb)
	defer c.Close()
	if err != nil {
		return nil, err
	}

	valueBytes, err = redis.Bytes(c.Do("Get", key))
	if err != nil {
		return nil, err
	}
	return valueBytes, nil
}

func (agent *Agent) Del(key string) (err error) {
	c, err := agent.selectDB(agent.RedisDb)
	defer c.Close()
	if err != nil {
		return err
	}

	_, err = c.Do("Del", key)
	if err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

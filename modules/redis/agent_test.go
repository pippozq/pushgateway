package redis

import "testing"

var (
	agent = new(Agent)
)

func TestAgent_Set(t *testing.T) {
	agent.RedisHost = "172.16.21.59"
	agent.RedisPassword = "reds"
	agent.RedisDb = "1"
	agent.RedisPort = "6379"
	agent.Pool = agent.InitPool()
	agent.Set("test", []byte("test expire"), "60")
}

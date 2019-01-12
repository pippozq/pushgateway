package redis

import "testing"

var (
	agent = new(Agent)
)

func TestAgent_Set(t *testing.T) {
	agent.RedisHost = "127.0.0.1"
	agent.RedisPassword = ""
	agent.RedisDb = 0
	agent.RedisPort = 6379
	agent.Pool = agent.InitPool()
	agent.Set("test", []byte("test expire"), 60)
}

package redis

import "testing"

var (
	agent = new(Agent)
)

func TestAgent_Set(t *testing.T) {
	agent.Pool = InitPool("172.16.21.59", "36379", "redis")
	agent.Set("test", []byte("test expire"), "9","60")
}

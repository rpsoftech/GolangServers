package whatsappdump_env

import "fmt"

func GetRedisKey(key string) string {
	return fmt.Sprintf("%s%s", Env.REDIS_DEFAULT_KEY, key)
}
func GetRedisEventKey(key string) string {
	return fmt.Sprintf("%s%s", Env.REDIS_DEFAULT_CHANNEL, key)
}

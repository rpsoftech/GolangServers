package dump_server_env

func GetRedisKey(key string) string {
	return Env.REDIS_DEFAULT_KEY + key
}
func GetRedisEventKey(key string) string {
	return Env.REDIS_DEFAULT_CHANNEL + key
}

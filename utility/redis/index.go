package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/events"
	"github.com/rpsoftech/golang-servers/validator"
)

type RedisClientConfig struct {
	REDIS_DB_HOST     string `json:"REDIS_DB_HOST" validate:"required"`
	REDIS_DB_PORT     int    `json:"REDIS_DB_PORT" validate:"required,port"`
	REDIS_DB_PASSWORD string `json:"REDIS_DB_PASSWORD" validate:"required"`
	REDIS_DB_DATABASE int    `json:"REDIS_DB_DATABASE" validate:"min=0,max=100"`
}

type RedisClientStruct struct {
	redisClient *redis.Client
}

var RedisClient *RedisClientStruct

var RedisCTX = context.Background()

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	// RedisClient.redisClient.Subscribe()
}

func InitRedisAndRedisClient() *RedisClientStruct {
	if RedisClient != nil {
		return RedisClient
	}
	redis_DB_DATABASE, err := strconv.Atoi(env.Env.GetEnv(env.REDIS_DB_DATABASE_KEY))
	if err != nil {
		// ... handle error
		panic(err)
	}
	redis_DB_PORT, err := strconv.Atoi(env.Env.GetEnv(env.REDIS_DB_PORT_KEY))
	if err != nil {
		// ... handle error
		panic(err)
	}
	config := &RedisClientConfig{
		REDIS_DB_PORT:     redis_DB_PORT,
		REDIS_DB_HOST:     env.Env.GetEnv(env.REDIS_DB_HOST_KEY),
		REDIS_DB_PASSWORD: env.Env.GetEnv(env.REDIS_DB_PASSWORD_KEY),
		REDIS_DB_DATABASE: redis_DB_DATABASE,
	}
	errs := validator.Validator.Validate(config)
	if len(errs) > 0 {
		println(errs)
		panic(errs[0])
	}
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%v:%d", config.REDIS_DB_HOST, config.REDIS_DB_PORT),
		Password: config.REDIS_DB_PASSWORD, // no password set
		DB:       config.REDIS_DB_DATABASE, // use default DB
	})

	RedisClient = &RedisClientStruct{
		redisClient: client,
	}
	go func() {
		res := RedisClient.redisClient.Ping(RedisCTX)
		if res.Err() != nil {
			panic(res.Err())
		}
	}()
	println("Redis Client Initialized")
	return RedisClient
}

func DeferFunction() {
	if err := RedisClient.redisClient.Close(); err != nil {
		panic(err)
	}
}

func (r *RedisClientStruct) SubscribeToChannels(channels ...string) *redis.PubSub {
	return r.redisClient.Subscribe(RedisCTX, channels...)
}

func (r *RedisClientStruct) PublishEvent(event *events.BaseEvent) {
	r.redisClient.Publish(RedisCTX, event.GetEventName(), event.GetPayloadString())
}
func (r *RedisClientStruct) GetHashValue(key string) map[string]string {
	return r.redisClient.HGetAll(RedisCTX, key).Val()
}
func (r *RedisClientStruct) GetStringData(key string) string {
	return r.redisClient.Get(RedisCTX, key).Val()
}

func (r *RedisClientStruct) RemoveKey(key ...string) {
	r.redisClient.Del(RedisCTX, key...)
}
func (r *RedisClientStruct) SetStringData(key string, value string, expiresIn int) {
	r.SetStringDataWithExpiry(key, value, time.Duration(expiresIn)*time.Second)
}
func (r *RedisClientStruct) SetStringDataWithExpiry(key string, value string, expiresIn time.Duration) {
	r.redisClient.Set(RedisCTX, key, value, expiresIn)
}

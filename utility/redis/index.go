package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/events"
	"github.com/rpsoftech/golang-servers/interfaces"
)

type RedisClientConfig struct {
	REDIS_DB_HOST     string `json:"REDIS_DB_HOST" validate:"required"`
	REDIS_DB_PORT     int    `json:"REDIS_DB_PORT" validate:"required,port"`
	REDIS_DB_PASSWORD string `json:"REDIS_DB_PASSWORD" validate:"required"`
	REDIS_DB_USERNAME string `json:"REDIS_DB_USERNAME"`
	REDIS_DB_DATABASE int    `json:"REDIS_DB_DATABASE" validate:"min=0,max=100"`
}

type RedisClientStruct struct {
	redisClient           *redis.Client
	REDIS_DEFAULT_KEY     string `json:"REDIS_DEFAULT_KEY"`
	REDIS_DEFAULT_CHANNEL string `json:"REDIS_DEFAULT_CHANNEL"`
}

const (
	TimeToLive_OneHour time.Duration = time.Hour
	TimeToLive_OneDay  time.Duration = time.Hour * 24
)

var (
	RedisClient *RedisClientStruct
	redisOnce   sync.Once
	RedisCTX    = context.Background()
)

func GetRedisClient() *RedisClientStruct {
	return InitRedisAndRedisClient()
}
func InitRedisAndRedisClient() *RedisClientStruct {
	redisOnce.Do(func() {
		redis_DB_DATABASE, err := strconv.Atoi(env.Env.GetEnv(env.REDIS_DB_DATABASE_KEY))
		if err != nil {
			panic(err)
		}
		redis_DB_PORT, err := strconv.Atoi(env.Env.GetEnv(env.REDIS_DB_PORT_KEY))
		if err != nil {
			panic(err)
		}

		config := &RedisClientConfig{
			REDIS_DB_PORT:     redis_DB_PORT,
			REDIS_DB_HOST:     env.Env.GetEnv(env.REDIS_DB_HOST_KEY),
			REDIS_DB_PASSWORD: env.Env.GetEnv(env.REDIS_DB_PASSWORD_KEY),
			REDIS_DB_DATABASE: redis_DB_DATABASE,
			REDIS_DB_USERNAME: env.Env.GetEnv(env.REDIS_DB_USERNAME_KEY),
		}
		env.ValidateEnv(config)

		client := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%v:%d", config.REDIS_DB_HOST, config.REDIS_DB_PORT),
			Password: config.REDIS_DB_PASSWORD,
			DB:       config.REDIS_DB_DATABASE,
			Username: config.REDIS_DB_USERNAME,
		})

		// Ping synchronously to guarantee the connection is alive
		if err := client.Ping(RedisCTX).Err(); err != nil {
			panic(fmt.Errorf("redis ping failed: %w", err))
		}
		defaultKey := env.Env.GetEnv(env.REDIS_DEFAULT_KEY_KEY)
		defaultChannel := env.Env.GetEnv(env.REDIS_DEFAULT_CHANNEL_KEY)
		RedisClient = &RedisClientStruct{
			redisClient:           client,
			REDIS_DEFAULT_KEY:     defaultKey,
			REDIS_DEFAULT_CHANNEL: defaultChannel,
		}
		println("Redis Client Initialized via sync.Once")
	})

	return RedisClient
}

func DeferFunction() {
	if err := RedisClient.redisClient.Close(); err != nil {
		panic(err)
	}
}

func CacheDataToRedis[TData interfaces.BaseEntityInterface](client *RedisClientStruct, entity *TData, key string, expiresIn time.Duration) error {
	entityStringBytes, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	entityString := string(entityStringBytes)
	client.SetStringDataWithExpiry(key, entityString, expiresIn)
	return nil
}

func (r *RedisClientStruct) SubscribeToChannels(channels ...string) *redis.PubSub {
	for chanel := range channels {
		channels[chanel] = r.GetRedisEventKey(channels[chanel])
	}
	return r.redisClient.Subscribe(RedisCTX, channels...)
}

func (r *RedisClientStruct) PublishEvent(event events.BaseEventInterface) {
	r.redisClient.Publish(RedisCTX, r.GetRedisEventKey(event.GetEventName()), event.GetPayloadString())
}
func (r *RedisClientStruct) PublishCustomEvent(event string, payload string) {
	r.redisClient.Publish(RedisCTX, r.GetRedisEventKey(event), payload)
}
func (r *RedisClientStruct) GetHashValue(key string) map[string]string {
	return r.redisClient.HGetAll(RedisCTX, r.GetRedisKey(key)).Val()
}
func (r *RedisClientStruct) GetStringDataCtx(ctx context.Context, key string) *redis.StringCmd {
	return r.redisClient.Get(ctx, r.GetRedisKey(key))
}
func (r *RedisClientStruct) GetStringData(key string) string {
	return r.GetStringDataCtx(RedisCTX, r.GetRedisKey(key)).Val()
}

func (r *RedisClientStruct) RemoveKey(key ...string) {
	for keyIndex := range key {
		key[keyIndex] = r.GetRedisKey(key[keyIndex])
	}
	r.redisClient.Del(RedisCTX, key...)
}
func (r *RedisClientStruct) SetStringData(key string, value string, expiresIn int) {
	r.SetStringDataWithExpiry(key, value, time.Duration(expiresIn)*time.Second)
}
func (r *RedisClientStruct) SetStringDataWithExpiry(key string, value string, expiresIn time.Duration) {
	r.SetStringDataWithExpiryCtx(RedisCTX, key, value, expiresIn)
}
func (r *RedisClientStruct) SetStringDataWithExpiryCtx(ctx context.Context, key string, value string, expiresIn time.Duration) {
	r.redisClient.Set(ctx, r.GetRedisKey(key), value, expiresIn)
}

func (r *RedisClientStruct) GetRedisKey(key string) string {
	return r.REDIS_DEFAULT_KEY + key
}
func (r *RedisClientStruct) GetRedisEventKey(key string) string {
	return r.REDIS_DEFAULT_CHANNEL + key
}

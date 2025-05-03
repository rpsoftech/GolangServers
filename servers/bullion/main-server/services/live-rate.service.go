package bullion_main_server_services

import (
	"encoding/json"
	"time"

	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

type liveRateServiceStruct struct {
	redisRepo   *redis.RedisClientStruct
	LastRateMap bullion_main_server_interfaces.LiveRateData
}

var LiveRateService *liveRateServiceStruct

func init() {
	service := getLiveRateService()
	service.lastRateReaderFromRedis()
	service.subscribeToRedisForRate()
}

func getLiveRateService() *liveRateServiceStruct {
	if LiveRateService == nil {
		LiveRateService = &liveRateServiceStruct{
			redisRepo:   redis.InitRedisAndRedisClient(),
			LastRateMap: make(bullion_main_server_interfaces.LiveRateData),
		}
		for _, k := range bullion_main_server_interfaces.SymbolsEnumArray {
			LiveRateService.LastRateMap[k] = make(map[bullion_main_server_interfaces.PriceKeyEnum]float64)
		}
		println("Live Rate Service Initialized")
	}
	return LiveRateService
}

func (s *liveRateServiceStruct) GetLastRate() *bullion_main_server_interfaces.LiveRateData {
	return &s.LastRateMap
}

func (s *liveRateServiceStruct) GetLiveRate(symbol bullion_main_server_interfaces.SymbolsEnum, priceKey bullion_main_server_interfaces.PriceKeyEnum) float64 {
	rateMap := s.LastRateMap[symbol]
	if len(rateMap) == 0 {
		return 0
	}
	return rateMap[priceKey]
}

func (s *liveRateServiceStruct) lastRateReaderFromRedis() {
	go func() {
		for {
			data := s.redisRepo.GetHashValue("LastRate")
			for keyString, value := range data {
				key := bullion_main_server_interfaces.SymbolsEnumFromString(keyString)
				if key != "" {
					symbolMap := s.LastRateMap[key]
					json.Unmarshal([]byte(value), &symbolMap)
				}
			}
			time.Sleep(15 * time.Second)
		}
	}()
}

// SubscribeToRedisForRate subscribes to the minirate Redis channel and
// updates the live rate service with the latest data from Redis.
func (s *liveRateServiceStruct) subscribeToRedisForRate() {
	psc := redis.RedisClient.SubscribeToChannels("minirate")

	go func() {
		// Listen to messages from the Redis channel
		for msg := range psc.Channel() {
			// Unmarshal the JSON data from the Redis message payload
			data := new(bullion_main_server_interfaces.LiveRateData)
			if err := json.Unmarshal([]byte(msg.Payload), data); err == nil {
				// Loop through each symbol in the data
				for symbol, rates := range *data {
					// If the symbol does not already exist in the live rate map
					// Add the symbol and its rates to the live rate map
					if _, ok := s.LastRateMap[symbol]; ok {
						for priceKey, v1 := range rates {
							s.LastRateMap[symbol][priceKey] = v1
						}
					}
				}
			}
		}
	}()
}

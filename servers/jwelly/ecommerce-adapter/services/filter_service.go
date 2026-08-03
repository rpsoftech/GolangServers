package ecommerce_services

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	ecommerce_dto "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/dto"
	ecommerce_repository "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/repository"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

type FilterService struct {
	filterRepo  *ecommerce_repository.FilterRepository
	redisClient *redis.RedisClientStruct
}

var (
	filterServiceOnce     sync.Once
	filterServiceInstance *FilterService
)

func GetFilterService() *FilterService {
	filterServiceOnce.Do(func() {
		filterServiceInstance = &FilterService{
			filterRepo:  ecommerce_repository.GetFilterRepository(),
			redisClient: redis.GetRedisClient(),
		}
	})
	return filterServiceInstance
}

func (s *FilterService) GetFilterOptions(ctx context.Context) (*ecommerce_dto.FilterOptionsDTO, error) {
	cacheKey := "products:filter_options"

	// 1. Check Redis Cache
	cachedData, err := s.redisClient.GetStringDataCtx(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		var result ecommerce_dto.FilterOptionsDTO
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			return &result, nil
		}
	}

	// 2. Cache Miss: Fetch from DB
	categories, err := s.filterRepo.FetchActiveCategories(ctx)
	if err != nil {
		return nil, err
	}

	purities, err := s.filterRepo.FetchActivePurities(ctx)
	if err != nil {
		return nil, err
	}

	result := &ecommerce_dto.FilterOptionsDTO{
		Categories: categories,
		Purities:   purities,
	}

	// 3. Store in Redis (1-hour TTL as master data rarely changes)
	if resultBytes, err := json.Marshal(result); err == nil {
		s.redisClient.SetStringDataWithExpiryCtx(ctx, cacheKey, string(resultBytes), 1*time.Hour)
	}

	return result, nil
}

func (s *FilterService) GetPuritiesOptions(ctx context.Context) (*[]ecommerce_dto.FilterOption, error) {
	cacheKey := "products:purities"

	// 1. Check Redis Cache
	cachedData, err := s.redisClient.GetStringDataCtx(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		var result []ecommerce_dto.FilterOption
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			return &result, nil
		}
	}

	purities, err := s.filterRepo.FetchActivePurities(ctx)
	if err != nil {
		return nil, err
	}

	// 3. Store in Redis (1-hour TTL as master data rarely changes)
	if resultBytes, err := json.Marshal(purities); err == nil {
		s.redisClient.SetStringDataWithExpiryCtx(ctx, cacheKey, string(resultBytes), 1*time.Hour)
	}

	return &purities, nil
}

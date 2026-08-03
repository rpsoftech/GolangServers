package ecommerce_services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	ecommerce_dto "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/dto"
	ecommerce_repository "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/repository"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

type CategoryService struct {
	categoryRepo *ecommerce_repository.CategoryRepository
	redisClient  *redis.RedisClientStruct
}

var (
	categoryServiceOnce     sync.Once
	categoryServiceInstance *CategoryService
)

func GetCategoryService() *CategoryService {
	categoryServiceOnce.Do(func() {
		categoryServiceInstance = &CategoryService{
			categoryRepo: ecommerce_repository.GetCategoryRepository(),
			redisClient:  redis.GetRedisClient(),
		}
	})
	return categoryServiceInstance
}

func (s *CategoryService) GetAllCategories(ctx context.Context) ([]ecommerce_dto.CategoryDTO, error) {
	cacheKey := "categories:all"

	// 1. Check Redis Cache
	cachedData, err := s.redisClient.GetStringDataCtx(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		var results []ecommerce_dto.CategoryDTO
		if err := json.Unmarshal([]byte(cachedData), &results); err == nil {
			return results, nil // Cache Hit
		}
	}

	// 2. Cache Miss: Execute DB Query
	rows, err := s.categoryRepo.FetchAllCategoriesRaw(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ecommerce_dto.CategoryDTO
	for rows.Next() {
		var groupId int
		var groupName string

		if err := rows.Scan(&groupId, &groupName); err != nil {
			log.Printf("Error scanning category row: %v", err)
			continue
		}

		results = append(results, ecommerce_dto.CategoryDTO{
			CategoryID:       fmt.Sprintf("%d", groupId),
			CategoryName:     groupName,
			ParentCategoryID: nil, // Defaulting to null as Wirewings handles the hierarchy
			ImageURL:         "",  // Managed directly on the Wirewings platform [cite: 162]
			SortOrder:        1,
			IsActive:         true,
		})
	}

	// 3. Mandatory loop error check
	if err := rows.Err(); err != nil {
		log.Printf("Error during category rows iteration: %v", err)
		return nil, err
	}

	if results == nil {
		results = []ecommerce_dto.CategoryDTO{}
	}

	// 4. Store Results in Redis (24-hour TTL for Categories)
	if resultBytes, err := json.Marshal(results); err == nil {
		s.redisClient.SetStringDataWithExpiryCtx(ctx, cacheKey, string(resultBytes), 24*time.Hour)
	}

	return results, nil
}

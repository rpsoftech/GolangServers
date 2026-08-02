package ecommerce_services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	ecommerce_dto "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/dto"
	ecommerce_repository "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/repository"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

type VariantService struct {
	variantRepo *ecommerce_repository.VariantRepository
	redisClient *redis.RedisClientStruct
}

var (
	variantServiceOnce     sync.Once
	variantServiceInstance *VariantService
)

func GetVariantService() *VariantService {
	variantServiceOnce.Do(func() {
		variantServiceInstance = &VariantService{
			variantRepo: ecommerce_repository.GetVariantRepository(),
			redisClient: redis.GetRedisClient(),
		}
	})
	return variantServiceInstance
}

// Extends VariantDTO to include Wirewings parent mapping
type WirewingsVariantDTO struct {
	ecommerce_dto.VariantDTO
	ProductID string `json:"product_id"`
	Purity    string `json:"purity,omitempty"`
}

func (s *VariantService) GetPaginatedVariants(ctx context.Context, limit, offset int) ([]WirewingsVariantDTO, error) {
	cacheKey := fmt.Sprintf("variants:all:limit:%d:offset:%d", limit, offset)

	// 1. Check Redis Cache
	cachedData, err := s.redisClient.GetStringDataCtx(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		var results []WirewingsVariantDTO
		if err := json.Unmarshal([]byte(cachedData), &results); err == nil {
			return results, nil // Cache Hit
		}
	}

	// 2. Cache Miss: Execute DB Query
	rows, err := s.variantRepo.FetchAllVariantsRaw(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []WirewingsVariantDTO

	for rows.Next() {
		var variantId, productId int
		var purity sql.NullString
		var gWt, nWt, tunch, wstg float64
		var status bool

		if err := rows.Scan(&variantId, &productId, &purity, &gWt, &nWt, &tunch, &wstg, &status); err != nil {
			log.Printf("Error scanning variant row: %v", err)
			continue
		}

		results = append(results, WirewingsVariantDTO{
			VariantDTO: ecommerce_dto.VariantDTO{
				VariantID:   variantId,
				GrossWeight: gWt,
				NetWeight:   nWt,
				VSellTunch:  tunch,
				VSellWstg:   wstg,
				IsActive:    status,
			},
			ProductID: fmt.Sprintf("%d", productId),
			Purity:    purity.String,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error during variant rows iteration: %v", err)
		return nil, err
	}

	if results == nil {
		results = []WirewingsVariantDTO{}
	}

	// 3. Store Results in Redis (15-min TTL)
	if resultBytes, err := json.Marshal(results); err == nil {
		s.redisClient.SetStringDataWithExpiryCtx(ctx, cacheKey, string(resultBytes), 15*time.Minute)
	}

	return results, nil
}

package ecommerce_services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	ecommerce_dto "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/dto"
	ecommerce_repository "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/repository"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

type ProductService struct {
	productRepo *ecommerce_repository.ProductRepository
	redisClient *redis.RedisClientStruct
}

var (
	productServiceOnce     sync.Once
	productServiceInstance *ProductService
)

func GetProductService() *ProductService {
	productServiceOnce.Do(func() {
		productServiceInstance = &ProductService{
			productRepo: ecommerce_repository.GetProductRepository(),
			redisClient: redis.GetRedisClient(),
		}
	})
	return productServiceInstance
}

func generateCacheKey(req interface{}, prefix string) string {
	reqBytes, _ := json.Marshal(req)
	hash := sha256.Sum256(reqBytes)
	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(hash[:]))
}

func (s *ProductService) GetAllProductsForWirewings(ctx context.Context, limit, offset int) ([]ecommerce_dto.ProductDTO, error) {
	cacheKey := fmt.Sprintf("products:all:limit:%d:offset:%d", limit, offset)

	cachedData, err := s.redisClient.GetStringDataCtx(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		var results []ecommerce_dto.ProductDTO
		if err := json.Unmarshal([]byte(cachedData), &results); err == nil {
			return results, nil
		}
	}

	rows, err := s.productRepo.FetchAllProductCatalogWithJSON(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ecommerce_dto.ProductDTO

	for rows.Next() {
		var tagId, groupId int
		var tag, itemName, groupName string
		var variantsJSON []byte
		var cretaedOn time.Time
		if err := rows.Scan(&tagId, &tag, &itemName, &groupId, &groupName, &cretaedOn, &variantsJSON); err != nil {
			log.Printf("Error scanning product row: %v", err)
			continue
		}

		var variants []*ecommerce_dto.VariantDTO
		if err := json.Unmarshal(variantsJSON, &variants); err != nil {
			continue
		}

		results = append(results, ecommerce_dto.ProductDTO{
			ProductID:      fmt.Sprintf("%d", tagId),
			SKU:            tag,
			ProductName:    itemName,
			CategoryID:     fmt.Sprintf("%d", groupId),
			CollectionName: groupName,
			ProductType:    "Gold",
			IsActive:       true,
			Variants:       variants,
		})
	}

	if results == nil {
		results = []ecommerce_dto.ProductDTO{}
	}

	if resultBytes, err := json.Marshal(results); err == nil {
		s.redisClient.SetStringDataWithExpiryCtx(ctx, cacheKey, string(resultBytes), 15*time.Minute)
	}

	return results, nil
}

func (s *ProductService) SearchAndFilterProducts(ctx context.Context, req *ecommerce_dto.ProductSearchRequest) ([]ecommerce_dto.ProductDTO, error) {
	cacheKey := generateCacheKey(req, "products:search")

	cachedData, err := s.redisClient.GetStringDataCtx(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		var results []ecommerce_dto.ProductDTO
		if err := json.Unmarshal([]byte(cachedData), &results); err == nil {
			return results, nil
		}
	}

	rows, err := s.productRepo.FetchFilteredProductsWithJSON(ctx, req)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ecommerce_dto.ProductDTO

	for rows.Next() {
		var tagId, groupId int
		var tag, itemName, groupName string
		var variantsJSON []byte
		var cretaedOn time.Time

		if err := rows.Scan(&tagId, &tag, &itemName, &groupId, &groupName, &cretaedOn, &variantsJSON); err != nil {
			log.Printf("Error scanning product filter row: %v", err)
			continue
		}

		var variants []*ecommerce_dto.VariantDTO
		if err := json.Unmarshal(variantsJSON, &variants); err != nil {
			continue
		}

		results = append(results, ecommerce_dto.ProductDTO{
			ProductID:      fmt.Sprintf("%d", tagId),
			SKU:            tag,
			ProductName:    itemName,
			CategoryID:     fmt.Sprintf("%d", groupId),
			CollectionName: groupName,
			ProductType:    "Gold",
			IsActive:       true,
			Variants:       variants,
		})
	}

	if results == nil {
		results = []ecommerce_dto.ProductDTO{}
	}

	if resultBytes, err := json.Marshal(results); err == nil {
		s.redisClient.SetStringDataWithExpiryCtx(ctx, cacheKey, string(resultBytes), 15*time.Minute)
	}

	return results, nil
}
func (s *ProductService) GetProductBySKU(ctx context.Context, sku string) (*ecommerce_dto.ProductDTO, error) {
	cacheKey := fmt.Sprintf("products:sku:%s", sku)

	// 1. Check Redis Cache
	cachedData, err := s.redisClient.GetStringDataCtx(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		var result ecommerce_dto.ProductDTO
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			return &result, nil // Cache Hit
		}
	}

	// 2. Cache Miss: Execute DB Query
	row := s.productRepo.FetchProductBySKUWithJSON(ctx, sku)

	var tagId, groupId int
	var tag, itemName, groupName string
	var variantsJSON []byte

	if err := row.Scan(&tagId, &tag, &itemName, &groupId, &groupName, &variantsJSON); err != nil {
		return nil, fmt.Errorf("product not found or error: %w", err)
	}

	var variants []*ecommerce_dto.VariantDTO
	if err := json.Unmarshal(variantsJSON, &variants); err != nil {
		return nil, fmt.Errorf("error parsing variants: %w", err)
	}

	result := &ecommerce_dto.ProductDTO{
		ProductID:      fmt.Sprintf("%d", tagId),
		SKU:            tag,
		ProductName:    itemName,
		CategoryID:     fmt.Sprintf("%d", groupId),
		CollectionName: groupName,
		ProductType:    "Gold",
		IsActive:       true,
		Variants:       variants,
	}

	// 3. Store Result in Redis (15-min TTL)
	if resultBytes, err := json.Marshal(result); err == nil {
		s.redisClient.SetStringDataWithExpiryCtx(ctx, cacheKey, string(resultBytes), 15*time.Minute)
	}

	return result, nil
}

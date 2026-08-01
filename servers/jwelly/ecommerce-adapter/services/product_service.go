package ecommerce_services

import (
	"context"
	"fmt"
	"log"

	ecommerce_dto "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/dto"
	ecommerce_repository "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/repository"
)

type ProductService struct{}

var productServiceInstance *ProductService

func GetProductService() *ProductService {
	if productServiceInstance == nil {
		productServiceInstance = &ProductService{}
	}
	return productServiceInstance
}

func (s *ProductService) GetAllProductsForWirewings(ctx context.Context, limit, offset int) ([]ecommerce_dto.ProductDTO, error) {
	repo := ecommerce_repository.GetProductRepository()
	rows, err := repo.FetchAllProductCatalogRaw(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Map to group variants by product tag
	productMap := make(map[string]*ecommerce_dto.ProductDTO)

	for rows.Next() {
		var tagId, groupId, variantId int
		var tag, itemName, groupName string
		var gWt, nWt, tunch, wstg float64
		var status bool

		if err := rows.Scan(&tagId, &tag, &itemName, &groupId, &groupName, &variantId, &gWt, &nWt, &tunch, &wstg, &status); err != nil {
			log.Printf("Error scanning product row: %v", err)
			continue
		}

		// If product doesn't exist in map yet, create it using the Tag as the parent SKU
		prod, exists := productMap[tag]
		if !exists {
			prod = &ecommerce_dto.ProductDTO{
				ProductID:      fmt.Sprintf("%d", tagId),
				SKU:            tag,
				ProductName:    itemName,
				CategoryID:     fmt.Sprintf("%d", groupId),
				CollectionName: groupName,
				ProductType:    "Gold",
				IsActive:       status,
				Variants:       []*ecommerce_dto.VariantDTO{},
			}
			productMap[tag] = prod
		}

		// Append the variation data
		prod.Variants = append(prod.Variants, &ecommerce_dto.VariantDTO{
			VariantID:   variantId,
			GrossWeight: gWt,
			NetWeight:   nWt,
			VSellTunch:  tunch,
			VSellWstg:   wstg,
			IsActive:    status,
		})
	}

	// Convert map to slice for JSON serialization
	var results []ecommerce_dto.ProductDTO
	for _, p := range productMap {
		results = append(results, *p)
	}

	return results, nil
}

func (s *ProductService) SearchAndFilterProducts(ctx context.Context, req *ecommerce_dto.ProductSearchRequest) ([]ecommerce_dto.ProductDTO, error) {
	repo := ecommerce_repository.GetProductRepository()
	rows, err := repo.FetchFilteredProducts(ctx, req)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	productMap := make(map[string]*ecommerce_dto.ProductDTO)

	for rows.Next() {
		var tagId, groupId, variantId int
		var tag, itemName, groupName string
		var gWt, nWt, tunch, wstg float64
		var status bool

		if err := rows.Scan(&tagId, &tag, &itemName, &groupId, &groupName, &variantId, &gWt, &nWt, &tunch, &wstg, &status); err != nil {
			log.Printf("Error scanning product filter row: %v", err)
			continue
		}

		prod, exists := productMap[tag]
		if !exists {
			prod = &ecommerce_dto.ProductDTO{
				ProductID:      fmt.Sprintf("%d", tagId),
				SKU:            tag,
				ProductName:    itemName,
				CategoryID:     fmt.Sprintf("%d", groupId),
				CollectionName: groupName,
				ProductType:    "Gold",
				IsActive:       status,
				Variants:       []*ecommerce_dto.VariantDTO{},
			}
			productMap[tag] = prod
		}

		prod.Variants = append(prod.Variants, &ecommerce_dto.VariantDTO{
			VariantID:   variantId,
			GrossWeight: gWt,
			NetWeight:   nWt,
			VSellTunch:  tunch,
			VSellWstg:   wstg,
			IsActive:    status,
		})
	}

	var results []ecommerce_dto.ProductDTO
	for _, p := range productMap {
		results = append(results, *p)
	}

	return results, nil
}

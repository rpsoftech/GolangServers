package ecommerce_repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
	ecommerce_dto "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/dto"
)

type ProductRepository struct {
	db *sql.DB
}

var (
	productRepoInstance *ProductRepository
	productRepoOnce     sync.Once
)

func GetProductRepository() *ProductRepository {
	productRepoOnce.Do(func() {
		productRepoInstance = &ProductRepository{
			db: ecommerce_env.MysqlConnections.Server.Db,
		}
	})
	return productRepoInstance
}

func (r *ProductRepository) FetchAllProductCatalogWithJSON(ctx context.Context, limit, offset int) (*sql.Rows, error) {
	query := `
		WITH PaginatedParents AS (
			SELECT itemTagId
			FROM ItemsTag
			ORDER BY itemTagId ASC
			LIMIT ? OFFSET ?
		)
		SELECT 
			t.itemTagId, 
			t.itemTag, 
			m.itemName, 
			g.itemGroupId, 
			g.itemGroup,
			JSON_ARRAYAGG(
				JSON_OBJECT(
					'variant_id', v.tagVariationId,
					'gross_weight', v.vGrossWt,
					'net_weight', v.vNetWt,
					'vSellTunch', v.vSellTunch,
					'vSellWstg', v.vSellWstg,
					'isActive', IF(v.vStatus = 1, true, false)
				)
			) AS variants
		FROM PaginatedParents pp
		JOIN ItemsTag t ON pp.itemTagId = t.itemTagId
		JOIN ItemMaster m ON t.tItemId = m.itemId
		JOIN ItemGroup g ON m.iGroupId = g.itemGroupId
		JOIN ItemTagVariation v ON t.itemTagId = v.vTagId
		WHERE v.vStatus = 1
		GROUP BY t.itemTagId, t.itemTag, m.itemName, g.itemGroupId, g.itemGroup
		ORDER BY t.itemTagId ASC;
	`
	return r.db.QueryContext(ctx, query, limit, offset)
}

func (r *ProductRepository) FetchFilteredProductsWithJSON(ctx context.Context, req *ecommerce_dto.ProductSearchRequest) (*sql.Rows, error) {
	var innerConditions []string
	var args []interface{}

	if req.Search != "" {
		innerConditions = append(innerConditions, "(m.itemName LIKE ? OR t.itemTag LIKE ?)")
		searchTerm := "%" + req.Search + "%"
		args = append(args, searchTerm, searchTerm)
	}

	if len(req.CategoryIDs) > 0 {
		placeholders := make([]string, len(req.CategoryIDs))
		for i, id := range req.CategoryIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		innerConditions = append(innerConditions, fmt.Sprintf("g.itemGroupId IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(req.CollectionIDs) > 0 {
		placeholders := make([]string, len(req.CollectionIDs))
		for i, id := range req.CollectionIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		innerConditions = append(innerConditions, fmt.Sprintf("g.itemGroupId IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(req.Purities) > 0 {
		placeholders := make([]string, len(req.Purities))
		for i, purity := range req.Purities {
			placeholders[i] = "?"
			args = append(args, purity)
		}
		innerConditions = append(innerConditions, fmt.Sprintf("v.vStampId IN (%s)", strings.Join(placeholders, ",")))
	}

	if req.WeightMin != nil {
		innerConditions = append(innerConditions, "v.vGrossWt >= ?")
		args = append(args, *req.WeightMin)
	}
	if req.WeightMax != nil {
		innerConditions = append(innerConditions, "v.vGrossWt <= ?")
		args = append(args, *req.WeightMax)
	}

	whereClause := "WHERE v.vStatus = 1"
	if len(innerConditions) > 0 {
		whereClause += " AND " + strings.Join(innerConditions, " AND ")
	}

	orderClause := "ORDER BY t.itemTagId ASC"
	switch req.SortBy {
	case "latest":
		orderClause = "ORDER BY t.tagCreatedDate DESC"
	case "weight_asc":
		orderClause = "ORDER BY MIN(v.vGrossWt) ASC"
	case "weight_desc":
		orderClause = "ORDER BY MAX(v.vGrossWt) DESC"
	case "name_asc":
		orderClause = "ORDER BY m.itemName ASC"
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	baseQuery := fmt.Sprintf(`
		WITH FilteredParents AS (
			SELECT t.itemTagId
			FROM ItemsTag t
			JOIN ItemMaster m ON t.tItemId = m.itemId
			JOIN ItemGroup g ON m.iGroupId = g.itemGroupId
			JOIN ItemTagVariation v ON t.itemTagId = v.vTagId
			%s
			GROUP BY t.itemTagId
			%s
			LIMIT ? OFFSET ?
		)
		SELECT 
			t.itemTagId, 
			t.itemTag, 
			m.itemName, 
			g.itemGroupId, 
			g.itemGroup,
			JSON_ARRAYAGG(
				JSON_OBJECT(
					'variant_id', v.tagVariationId,
					'gross_weight', v.vGrossWt,
					'net_weight', v.vNetWt,
					'vSellTunch', v.vSellTunch,
					'vSellWstg', v.vSellWstg,
					'isActive', IF(v.vStatus = 1, true, false)
				)
			) AS variants
		FROM FilteredParents fp
		JOIN ItemsTag t ON fp.itemTagId = t.itemTagId
		JOIN ItemMaster m ON t.tItemId = m.itemId
		JOIN ItemGroup g ON m.iGroupId = g.itemGroupId
		JOIN ItemTagVariation v ON t.itemTagId = v.vTagId
		WHERE v.vStatus = 1
		GROUP BY t.itemTagId, t.itemTag, m.itemName, g.itemGroupId, g.itemGroup
		%s;
	`, whereClause, orderClause, orderClause)

	args = append(args, limit, offset)

	return r.db.QueryContext(ctx, baseQuery, args...)
}

func (r *ProductRepository) FetchProductBySKUWithJSON(ctx context.Context, sku string) *sql.Row {
	query := `
		SELECT 
			t.itemTagId, 
			t.itemTag, 
			m.itemName, 
			g.itemGroupId, 
			g.itemGroup,
			JSON_ARRAYAGG(
				JSON_OBJECT(
					'variant_id', v.tagVariationId,
					'gross_weight', v.vGrossWt,
					'net_weight', v.vNetWt,
					'vSellTunch', v.vSellTunch,
					'vSellWstg', v.vSellWstg,
					'isActive', IF(v.vStatus = 1, true, false)
				)
			) AS variants
		FROM ItemsTag t
		JOIN ItemMaster m ON t.tItemId = m.itemId
		JOIN ItemGroup g ON m.iGroupId = g.itemGroupId
		JOIN ItemTagVariation v ON t.itemTagId = v.vTagId
		WHERE v.vStatus = 1 AND t.itemTag = ?
		GROUP BY t.itemTagId, t.itemTag, m.itemName, g.itemGroupId, g.itemGroup;
	`
	return r.db.QueryRowContext(ctx, query, sku)
}

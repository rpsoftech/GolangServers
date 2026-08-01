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
	db                       *sql.DB
	stmtGetAllProductCatalog *sql.Stmt
}

var (
	productRepoInstance *ProductRepository
	productRepoOnce     sync.Once
)

func GetProductRepository() *ProductRepository {
	productRepoOnce.Do(func() {
		db := ecommerce_env.MysqlConnections.Server.Db
		query := `
			SELECT 
				t.itemTagId, t.itemTag, m.itemName, g.itemGroupId, g.itemGroup,
				v.tagVariationId, v.vGrossWt, v.vNetWt, v.vSellTunch, v.vSellWstg, v.vStatus
			FROM ItemsTag t
			JOIN ItemMaster m ON t.tItemId = m.itemId
			JOIN ItemGroup g ON m.iGroupId = g.itemGroupId
			JOIN ItemTagVariation v ON t.itemTagId = v.vTagId
			WHERE v.vStatus = 1
			LIMIT ? OFFSET ?;
		`

		stmt, err := db.Prepare(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to pre-compile Product Catalog query: %v", err))
		}

		productRepoInstance = &ProductRepository{
			db:                       db,
			stmtGetAllProductCatalog: stmt,
		}
	})
	return productRepoInstance
}
func (r *ProductRepository) FetchAllProductCatalogRaw(ctx context.Context, limit, offset int) (*sql.Rows, error) {
	return r.stmtGetAllProductCatalog.QueryContext(ctx, limit, offset)
}

// FetchFilteredProducts dynamically constructs a safe query based on Wirewings filter requirements
func (r *ProductRepository) FetchFilteredProducts(ctx context.Context, req *ecommerce_dto.ProductSearchRequest) (*sql.Rows, error) {
	baseQuery := `
		SELECT 
			t.itemTagId, t.itemTag, m.itemName, g.itemGroupId, g.itemGroup,
			v.tagVariationId, v.vGrossWt, v.vNetWt, v.vSellTunch, v.vSellWstg, v.vStatus
		FROM ItemsTag t
		JOIN ItemMaster m ON t.tItemId = m.itemId
		JOIN ItemGroup g ON m.iGroupId = g.itemGroupId
		JOIN ItemTagVariation v ON t.itemTagId = v.vTagId
		WHERE v.vStatus = 1
	`

	var conditions []string
	var args []interface{}

	// Keyword Search filter
	if req.Search != "" {
		conditions = append(conditions, "(m.itemName LIKE ? OR t.itemTag LIKE ?)")
		searchTerm := "%" + req.Search + "%"
		args = append(args, searchTerm, searchTerm)
	}

	// Category IDs filter[cite: 1]
	if len(req.CategoryIDs) > 0 {
		placeholders := make([]string, len(req.CategoryIDs))
		for i, id := range req.CategoryIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		conditions = append(conditions, fmt.Sprintf("g.itemGroupId IN (%s)", strings.Join(placeholders, ",")))
	}

	// Weight Range filters[cite: 1]
	if req.WeightMin != nil {
		conditions = append(conditions, "v.vGrossWt >= ?")
		args = append(args, *req.WeightMin)
	}
	if req.WeightMax != nil {
		conditions = append(conditions, "v.vGrossWt <= ?")
		args = append(args, *req.WeightMax)
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Sorting logic[cite: 1]
	if req.SortBy == "latest" {
		baseQuery += " ORDER BY t.tagCreatedDate DESC"
	} else {
		baseQuery += " ORDER BY t.itemTagId ASC"
	}

	// Pagination[cite: 1]
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	baseQuery += " LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	return r.db.QueryContext(ctx, baseQuery, args...)
}

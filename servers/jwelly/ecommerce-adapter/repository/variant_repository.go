package ecommerce_repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
)

type VariantRepository struct {
	db                 *sql.DB
	stmtGetAllVariants *sql.Stmt
}

var (
	variantRepoInstance *VariantRepository
	variantRepoOnce     sync.Once
)

func GetVariantRepository() *VariantRepository {
	variantRepoOnce.Do(func() {
		db := ecommerce_env.MysqlConnections.Server.Db

		// Fetch variants linked to their parent product tags
		query := `
			SELECT 
				v.tagVariationId AS variant_id,
				t.itemTagId AS product_id,
				v.vStampId AS purity,
				v.vGrossWt,
				v.vNetWt,
				v.vSellTunch,
				v.vSellWstg,
				v.vStatus
			FROM ItemTagVariation v
			JOIN ItemsTag t ON v.vTagId = t.itemTagId
			WHERE v.vStatus = 1
			ORDER BY v.tagVariationId ASC
			LIMIT ? OFFSET ?;
		`

		stmt, err := db.Prepare(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to pre-compile Variant query: %v", err))
		}

		variantRepoInstance = &VariantRepository{
			db:                 db,
			stmtGetAllVariants: stmt,
		}
	})
	return variantRepoInstance
}

func (r *VariantRepository) FetchAllVariantsRaw(ctx context.Context, limit, offset int) (*sql.Rows, error) {
	return r.stmtGetAllVariants.QueryContext(ctx, limit, offset)
}

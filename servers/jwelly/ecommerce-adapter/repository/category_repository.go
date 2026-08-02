package ecommerce_repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
)

type CategoryRepository struct {
	db                   *sql.DB
	stmtGetAllCategories *sql.Stmt
}

var (
	categoryRepoInstance *CategoryRepository
	categoryRepoOnce     sync.Once
)

func GetCategoryRepository() *CategoryRepository {
	categoryRepoOnce.Do(func() {
		db := ecommerce_env.MysqlConnections.Server.Db

		query := `SELECT itemGroupId, itemGroup FROM ItemGroup ORDER BY itemGroup ASC;`
		stmt, err := db.Prepare(query)
		if err != nil {
			panic(fmt.Sprintf("Failed to pre-compile Category query: %v", err))
		}

		categoryRepoInstance = &CategoryRepository{
			db:                   db,
			stmtGetAllCategories: stmt,
		}
	})
	return categoryRepoInstance
}

func (r *CategoryRepository) FetchAllCategoriesRaw(ctx context.Context) (*sql.Rows, error) {
	return r.stmtGetAllCategories.QueryContext(ctx)
}

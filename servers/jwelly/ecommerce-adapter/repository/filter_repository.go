package ecommerce_repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
	ecommerce_dto "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/dto"
)

type FilterRepository struct {
	db                 *sql.DB
	stmtGetAllPurities *sql.Stmt
}

var (
	filterRepoInstance *FilterRepository
	filterRepoOnce     sync.Once
)

func GetFilterRepository() *FilterRepository {
	filterRepoOnce.Do(func() {
		db := ecommerce_env.MysqlConnections.Server.Db
		queryPurities := `SELECT stampId, STAMP FROM Stamp ORDER BY STAMP ASC;`
		stmtPurities, err := db.Prepare(queryPurities)
		if err != nil {
			panic(fmt.Sprintf("Failed to pre-compile Purities query: %v", err))
		}
		filterRepoInstance = &FilterRepository{
			db:                 db,
			stmtGetAllPurities: stmtPurities,
		}
	})
	return filterRepoInstance
}

func (r *FilterRepository) FetchActiveCategories(ctx context.Context) ([]ecommerce_dto.FilterOption, error) {
	query := `SELECT DISTINCT itemGroupId, itemGroup FROM ItemGroup ORDER BY itemGroup ASC;`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []ecommerce_dto.FilterOption
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err == nil {
			categories = append(categories, ecommerce_dto.FilterOption{ID: id, Name: name})
		}
	}
	// Check for errors encountered during iteration
	if err := rows.Err(); err != nil {
		log.Printf("Error during rows iteration: %v", err)
		return nil, err
	}
	return categories, nil
}

func (r *FilterRepository) FetchActivePurities(ctx context.Context) ([]ecommerce_dto.FilterOption, error) {
	// Execute the pre-compiled statement
	rows, err := r.stmtGetAllPurities.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var purities []ecommerce_dto.FilterOption
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err == nil {
			purities = append(purities, ecommerce_dto.FilterOption{ID: id, Name: name})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return purities, nil
}

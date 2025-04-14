package surrealdb

import (
	"fmt"

	"github.com/rpsoftech/golang-servers/env"
	surrealdb "github.com/surrealdb/surrealdb.go"
)

type SurrealdbConfig struct {
	SURREAL_URL         string `json:"SURREAL_URL" validate:"required,url"`
	SURREAL_NAMESPACE   string `json:"SURREAL_NAMESPACE" validate:"required,min=3"`
	SURREAL_DB_NAME     string `json:"SURREAL_DB_NAME" validate:"required,min=3"`
	SURREAL_DB_USERNAME string `json:"SURREAL_DB_USERNAME" validate:"required"`
	SURREAL_DB_PASSWORD string `json:"SURREAL_DB_PASSWORD" validate:"required"`
}

type SurrealDBStruct struct {
	db *surrealdb.DB
}

// var token = ""
var SurrealDb *SurrealDBStruct

func DeferFunction() {

	if err := SurrealDb.db.Invalidate(); err != nil {
		panic(err)
	}
	fmt.Println("Surrealdb Defering...")
}
func init() {
	fmt.Println("Surrealdb Initalizing...")
	config := &SurrealdbConfig{
		SURREAL_URL:         env.Env.GetEnv(env.SURREAL_URL_KEY),
		SURREAL_NAMESPACE:   env.Env.GetEnv(env.SURREAL_NAMESPACE_KEY),
		SURREAL_DB_NAME:     env.Env.GetEnv(env.SURREAL_DB_NAME_KEY),
		SURREAL_DB_USERNAME: env.Env.GetEnv(env.SURREAL_DB_USERNAME_KEY),
		SURREAL_DB_PASSWORD: env.Env.GetEnv(env.SURREAL_DB_PASSWORD_KEY),
	}
	db, err := surrealdb.New(config.SURREAL_URL)
	if err != nil {
		panic(err)
	}

	// Set the namespace and database
	if err = db.Use(config.SURREAL_NAMESPACE, config.SURREAL_DB_NAME); err != nil {
		panic(err)
	}

	// Sign in to authentication `db`
	authData := &surrealdb.Auth{
		Username: config.SURREAL_DB_USERNAME, // use your setup username
		Password: config.SURREAL_DB_PASSWORD, // use your setup password
	}
	token, err := db.SignIn(authData)
	if err != nil {
		panic(err)
	}
	if err := db.Authenticate(token); err != nil {
		panic(err)
	}
	SurrealDb = &SurrealDBStruct{db: db}
	fmt.Println("Surrealdb Initalized")
}

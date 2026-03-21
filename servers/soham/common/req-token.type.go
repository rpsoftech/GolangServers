package soham_common_req_keys

import "github.com/golang-jwt/jwt/v5"

type ReqTokenType struct {
	*jwt.RegisteredClaims
	From int `json:"from" validate:"required"`
}

package jwt

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rpsoftech/golang-servers/interfaces"
)

var (
	ErrTokenExpired           = jwt.ErrTokenExpired
	ErrInvalidToken           = errors.New("invalid token")
	ErrJwt                    = errors.New("something went wrong with Token")
	ErrSignatureInvalid       = jwt.ErrSignatureInvalid
	ErrInvalidSignatureMethod = errors.New("unexpected signing method")
)

type GeneralPurposeTokenGeneration struct {
	*jwt.RegisteredClaims
	ExtraClaim map[string]any `json:"claims,omitempty"`
}
type TokenService struct {
	SigningKey []byte
}

func GenerateToken[IClaims jwt.Claims](t *TokenService, claims IClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.SigningKey)
}

func (t *TokenService) GenerateToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.SigningKey)
}

//	func VerifyTokenWithClaims[TClaims GeneralPurposeTokenGeneration](t *TokenService, token string, claim *GeneralPurposeTokenGeneration) error {
//		claimRaw, err := t.verifyTokenReturnRaw(&token)
//		if err != nil {
//			return err
//		}
//		*claim = claimRaw.Claims
//		return nil
//	}
func (t *TokenService) VerifyTokenWithClaims(token string, claim *any) error {
	claimRaw, err := t.verifyTokenReturnRaw(&token)
	if err != nil {
		return err
	}
	*claim = claimRaw.Claims
	return nil
}

func (t *TokenService) verifyTokenReturnRaw(token *string) (*jwt.Token, error) {
	claimRaw, err := jwt.ParseWithClaims(*token, &GeneralPurposeTokenGeneration{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignatureMethod
		}
		return t.SigningKey, nil
	})
	if err != nil {
		if errors.Is(err, ErrTokenExpired) {
			err = &interfaces.RequestError{
				StatusCode: 403,
				Code:       interfaces.ERROR_TOKEN_EXPIRED,
				Message:    "Token Expired",
				Name:       "ERROR_TOKEN_EXPIRED",
				Extra:      err,
			}
		} else if errors.Is(err, ErrInvalidSignatureMethod) {
			err = &interfaces.RequestError{
				StatusCode: 401,
				Code:       interfaces.ERROR_INVALID_TOKEN_SIGNATURE_METHOD,
				Message:    "Invalid Signature Method",
				Name:       "ERROR_INVALID_TOKEN_SIGNATURE",
				Extra:      ErrInvalidSignatureMethod,
			}
		} else if errors.Is(err, ErrSignatureInvalid) {
			err = &interfaces.RequestError{
				StatusCode: 401,
				Code:       interfaces.ERROR_INVALID_TOKEN_SIGNATURE,
				Message:    "Invalid TOKEN Signature",
				Name:       "ERROR_INVALID_TOKEN_SIGNATURE",
				Extra:      ErrInvalidSignatureMethod,
			}
		} else {
			err = &interfaces.RequestError{
				StatusCode: 401,
				Code:       interfaces.ERROR_INTERNAL_SERVER,
				Message:    "Error In Token Parsing",
				Name:       "ERROR_IN_TOKEN_PARSING",
				Extra:      err,
			}
		}
		return nil, err
	}
	if !claimRaw.Valid {
		return nil, &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Error InValid Token",
			Name:       "ERROR_INVALID_TOKEN",
		}
	}
	return claimRaw, nil
}

func VerifyToken[IClaims jwt.Claims](t *TokenService, token *string) (*IClaims, error) {
	claimRaw, err := t.verifyTokenReturnRaw(token)
	if err != nil {
		return nil, err
	}
	claim, ok := claimRaw.Claims.(IClaims)
	if !ok {
		err = &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Error InValid Token Body",
			Name:       "ERROR_INVALID_TOKEN_BODY",
			Extra:      err,
		}
	}

	return &claim, err
}

func (t *TokenService) VerifyToken(token *string) (*GeneralPurposeTokenGeneration, error) {
	claimRaw, err := t.verifyTokenReturnRaw(token)
	if err != nil {
		return nil, err
	}
	claim, ok := claimRaw.Claims.(*GeneralPurposeTokenGeneration)
	if !ok {
		err = &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Error InValid Token Body",
			Name:       "ERROR_INVALID_TOKEN_BODY",
			Extra:      err,
		}
	}

	return claim, err
}

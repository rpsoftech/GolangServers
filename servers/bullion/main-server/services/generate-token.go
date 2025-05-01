package bullion_main_server_services

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rpsoftech/golang-servers/interfaces"
	localJwt "github.com/rpsoftech/golang-servers/utility/jwt"

	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

func generateTokens(userId string, bullionId string, role bullion_main_server_interfaces.UserRoles) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	var tokenResponse *bullion_main_server_interfaces.TokenResponseBody
	now := time.Now()
	accessToken, err := AccessTokenService.GenerateToken(bullion_main_server_interfaces.GeneralUserAccessRefreshToken{
		UserId:    userId,
		BullionId: bullionId,
		Role:      role,
		GeneralPurposeTokenGeneration: &localJwt.GeneralPurposeTokenGeneration{
			RegisteredClaims: &jwt.RegisteredClaims{
				IssuedAt:  &jwt.NumericDate{Time: now},
				ExpiresAt: &jwt.NumericDate{Time: now.Add(time.Minute * 30)},
			},
		},
	})
	if err != nil {
		err = &bullion_main_server_interfaces.RequestError{
			Code:    interfaces.ERROR_INTERNAL_SERVER,
			Message: "JWT ACCESS TOKEN GENERATION ERROR",
			Name:    "ERROR_INTERNAL_ERROR",
			Extra:   err,
		}
		return tokenResponse, err
	}
	refreshToken, err := RefreshTokenService.GenerateToken(bullion_main_server_interfaces.GeneralUserAccessRefreshToken{
		UserId:    userId,
		BullionId: bullionId,
		Role:      role,
		GeneralPurposeTokenGeneration: &localJwt.GeneralPurposeTokenGeneration{
			RegisteredClaims: &jwt.RegisteredClaims{
				IssuedAt: &jwt.NumericDate{Time: now},
				// ExpiresAt: &jwt.NumericDate{Time: now.Add(time.Hour * 24 * 30)},
			},
		},
	})
	if err != nil {
		err = &bullion_main_server_interfaces.RequestError{
			Code:    interfaces.ERROR_INTERNAL_SERVER,
			Message: "JWT ACCESS TOKEN GENERATION ERROR",
			Name:    "ERROR_INTERNAL_ERROR",
			Extra:   err,
		}
		return tokenResponse, err
	}

	firebaseToken, err := FirebaseAuthService.GenerateCustomToken(userId, map[string]interface{}{
		"userId":    userId,
		"bullionId": bullionId,
		"role":      role,
	})
	tokenResponse = &bullion_main_server_interfaces.TokenResponseBody{
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		FirebaseToken: firebaseToken,
	}
	return tokenResponse, err
}

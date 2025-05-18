package bullion_main_server_services

import (
	"firebase.google.com/go/v4/auth"
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/utility/firebase"
)

type firebaseAuthService struct {
	auth *auth.Client
}

var FirebaseAuthService *firebaseAuthService

func init() {
	FirebaseAuthService = &firebaseAuthService{
		auth: firebase.FirebaseAuth,
	}
	println("Firebase Auth Service Initialized")
}

func (s *firebaseAuthService) GenerateCustomToken(uid string, claims map[string]interface{}) (string, error) {
	token, err := s.auth.CustomTokenWithClaims(firebase.FirebaseCtx, uid, claims)
	if err != nil {
		err = &interfaces.RequestError{
			StatusCode: 500,
			Code:       interfaces.ERROR_INTERNAL_SERVER, Message: "Issue In Generating Firebase Token",
			Name: "INTERNAL_SERVER_ERROR", Extra: err,
		}
		return token, err
	}
	return token, err
}

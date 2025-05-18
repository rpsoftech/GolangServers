package bullion_main_server_services

import (
	"net/http"

	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_repos "github.com/rpsoftech/golang-servers/servers/bullion/main-server/repos"
)

type adminUserService struct {
	adminUserRepo *bullion_main_server_repos.AdminUserRepoStruct
}

var AdminUserService *adminUserService

func init() {
	AdminUserService = &adminUserService{
		adminUserRepo: bullion_main_server_repos.AdminUserRepo,
	}
	println("Admin Service Initialized")
}

func (service *adminUserService) ValidateUserAndGenerateToken(uname string, password string, bullionId string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	admin, err := service.adminUserRepo.FindOneUserNameAndBullionId(uname, bullionId)
	if err != nil {
		return nil, err
	}
	if !admin.MatchPassword(password) {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusUnauthorized,
			Code:       interfaces.ERROR_INVALID_PASSWORD,
			Message:    "Invalid Password",
			Name:       "ERROR_INVALID_PASSWORD",
		}
	}
	return generateTokens(admin.ID, bullionId, bullion_main_server_interfaces.ROLE_ADMIN)
}

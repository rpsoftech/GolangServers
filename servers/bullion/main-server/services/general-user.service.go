package bullion_main_server_services

import (
	"fmt"
	"net/http"

	"github.com/go-faker/faker/v4"
	"github.com/mitchellh/mapstructure"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_repos "github.com/rpsoftech/golang-servers/servers/bullion/main-server/repos"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/utility/jwt"
	"github.com/rpsoftech/golang-servers/validator"
)

type generalUserService struct {
	generalUserReqRepo  *bullion_main_server_repos.GeneralUserReqRepoStruct
	GeneralUserRepo     *bullion_main_server_repos.GeneralUserRepoStruct
	BullionSiteInfoRepo *bullion_main_server_repos.BullionSiteInfoRepoStruct
}

var GeneralUserService *generalUserService

func init() {
	GeneralUserService = &generalUserService{
		GeneralUserRepo:     bullion_main_server_repos.GeneralUserRepo,
		BullionSiteInfoRepo: bullion_main_server_repos.BullionSiteInfoRepo,
		generalUserReqRepo:  bullion_main_server_repos.GeneralUserReqRepo,
	}
	println("General User Service Initialized")
}

func (service *generalUserService) RegisterNew(bullionId string, user interface{}) (*bullion_main_server_interfaces.GeneralUserEntity, error) {
	Bullion, err := service.BullionSiteInfoRepo.FindOne(bullionId)
	if err != nil {
		return nil, err
	}

	baseGeneralUser := bullion_main_server_interfaces.GeneralUser{
		IsAuto: false,
	}

	if Bullion.GeneralUserInfo.AutoLogin {
		baseGeneralUser = bullion_main_server_interfaces.GeneralUser{
			FirstName:     faker.FirstName(),
			LastName:      faker.LastName(),
			FirmName:      faker.Username(),
			ContactNumber: faker.Phonenumber(),
			GstNumber:     validator.GenerateRandomGstNumber(),
			OS:            "AUTO",
			IsAuto:        true,
			DeviceId:      faker.UUIDDigit(),
			DeviceType:    bullion_main_server_interfaces.DEVICE_TYPE_IOS,
		}
	}

	baseGeneralUser.RandomPass = faker.Password()

	err = mapstructure.Decode(user, &baseGeneralUser)
	if err != nil {
		return nil, err
	}

	if err := utility_functions.ValidateReqInput(&baseGeneralUser); err != nil {
		return nil, err
	}

	entity := bullion_main_server_interfaces.CreateNewGeneralUser(baseGeneralUser)
	if err := utility_functions.ValidateReqInput(entity); err != nil {
		return nil, err
	}

	_, err = service.GeneralUserRepo.Save(entity)
	if err != nil {
		return nil, err
	}

	_, err = service.sendApprovalRequest(entity, Bullion)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (service *generalUserService) CreateApprovalRequest(userId string, password string, bullionId string) (*bullion_main_server_interfaces.GeneralUserReqEntity, error) {
	userEntity, err := service.GetGeneralUserDetailsByIdPassword(userId, password)
	if err != nil {
		return nil, err
	}

	bullionEntity, err := service.BullionSiteInfoRepo.FindOne(bullionId)
	if err != nil {
		return nil, err
	}

	return service.sendApprovalRequest(userEntity, bullionEntity)
}
func (service *generalUserService) sendApprovalRequest(user *bullion_main_server_interfaces.GeneralUserEntity, bullion *bullion_main_server_interfaces.BullionSiteInfoEntity) (*bullion_main_server_interfaces.GeneralUserReqEntity, error) {
	existingReq, err := service.generalUserReqRepo.FindOneByGeneralUserIdAndBullionId(user.ID, bullion.ID)
	if err == nil {
		if existingReq != nil {
			return nil, &bullion_main_server_interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_GENERAL_USER_REQ_EXISTS,
				Message:    "REQUEST ALREADY EXISTS",
				Name:       "ERROR_GENERAL_USER_REQ_EXISTS",
			}
		} else {
			return nil, &bullion_main_server_interfaces.RequestError{
				StatusCode: 500,
				Code:       interfaces.ERROR_INTERNAL_SERVER,
				Message:    "REQUEST CHECK ERROR",
				Name:       "ERROR_GENERAL_USER_REQ_EXISTS",
			}
		}
	}

	reqEntity := bullion_main_server_interfaces.CreateNewGeneralUserReq(user.ID, bullion.ID, bullion_main_server_interfaces.GENERAL_USER_AUTH_STATUS_REQUESTED)
	if bullion.GeneralUserInfo.AutoApprove {
		reqEntity.Status = bullion_main_server_interfaces.GENERAL_USER_AUTH_STATUS_AUTHORIZED
	}

	return service.generalUserReqRepo.Save(reqEntity)
}
func (service *generalUserService) GetGeneralUserDetailsByIdPassword(id string, password string) (*bullion_main_server_interfaces.GeneralUserEntity, error) {
	entity, err := service.GeneralUserRepo.FindOne(id)
	if err != nil {
		return entity, err
	}
	if entity.RandomPass != password {
		err = &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_GENERAL_USER_INVALID_PASSWORD,
			Message:    fmt.Sprintf("GeneralUser Entity invalid password %s ", password),
			Name:       "ERROR_GENERAL_USER_INVALID_PASSWORD",
		}
		return entity, err
	}
	return entity, err
}

func (service *generalUserService) ValidateApprovalAndGenerateToken(userId string, password string, bullionId string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	userEntity, err := service.GetGeneralUserDetailsByIdPassword(userId, password)
	if err != nil {
		return nil, err
	}
	return service.validateApprovalAndGenerateTokenStage2(userEntity, bullionId)
}
func (service *generalUserService) validateApprovalAndGenerateTokenStage2(user *bullion_main_server_interfaces.GeneralUserEntity, bullionId string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	reqEntity, err := service.generalUserReqRepo.FindOneByGeneralUserIdAndBullionId(user.ID, bullionId)
	if err != nil || reqEntity == nil {
		return nil, &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_GENERAL_USER_REQ_NOT_FOUND,
			Message:    "REQUEST DOES NOT EXISTS",
			Name:       "ERROR_GENERAL_USER_REQ_NOT_FOUND",
		}
	}

	switch reqEntity.Status {
	case bullion_main_server_interfaces.GENERAL_USER_AUTH_STATUS_REQUESTED:
		return nil, &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_GENERAL_USER_REQ_PENDING,
			Message:    "REQUEST PENDING",
			Name:       "ERROR_GENERAL_USER_REQ_PENDING",
		}
	case bullion_main_server_interfaces.GENERAL_USER_AUTH_STATUS_REJECTED:
		return nil, &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_GENERAL_USER_REQ_REJECTED,
			Message:    "REQUEST REJECTED",
			Name:       "ERROR_GENERAL_USER_REQ_REJECTED",
		}
	case bullion_main_server_interfaces.GENERAL_USER_AUTH_STATUS_AUTHORIZED:
		return service.generateTokens(user.ID, bullionId)
	default:
		return nil, &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_GENERAL_USER_INVALID_STATUS,
			Message:    "Invalid Request Status",
			Name:       "ERROR_GENERAL_USER_INVALID_STATUS",
		}
	}
}

func (service *generalUserService) generateTokens(userId string, bullionId string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	return generateTokens(userId, bullionId, bullion_main_server_interfaces.ROLE_GENERAL_USER)
}
func (service *generalUserService) RefreshToken(token string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	var tokenResponse *bullion_main_server_interfaces.TokenResponseBody
	tokenBody, err := jwt.VerifyToken[bullion_main_server_interfaces.GeneralUserAccessRefreshToken](RefreshTokenService, &token)
	// RefreshTokenService.VerifyToken(token)
	if err != nil {
		return tokenResponse, err
	}

	generalUserEntity, err := service.GeneralUserRepo.FindOne(tokenBody.UserId)
	if err != nil {
		return tokenResponse, err
	}
	return service.validateApprovalAndGenerateTokenStage2(generalUserEntity, tokenBody.BullionId)
}

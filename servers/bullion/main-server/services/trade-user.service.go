package bullion_main_server_services

import (
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mitchellh/mapstructure"

	// localJwt "github.com/rpsoftech/bullion-server/src/utility/jwt"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_events "github.com/rpsoftech/golang-servers/servers/bullion/main-server/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_repos "github.com/rpsoftech/golang-servers/servers/bullion/main-server/repos"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	localJwt "github.com/rpsoftech/golang-servers/utility/jwt"
)

type tradeUserServiceStruct struct {
	accessTokenService *localJwt.TokenService
	tradeUserRepo      *bullion_main_server_repos.TradeUserRepoStruct
	eventBus           *eventBusService
	bullionService     *bullionDetailsService
	firebaseDb         *firebaseDatabaseService
	sendMsgService     *sendMsgService
	realtimeDatabase   *firebaseDatabaseService
}

var TradeUserService *tradeUserServiceStruct

func init() {
	getTradeUserService()
}

func getTradeUserService() *tradeUserServiceStruct {
	if TradeUserService == nil {
		TradeUserService = &tradeUserServiceStruct{
			tradeUserRepo:      bullion_main_server_repos.TradeUserRepo,
			accessTokenService: AccessTokenService,
			eventBus:           getEventBusService(),
			firebaseDb:         getFirebaseRealTimeDatabase(),
			sendMsgService:     getSendMsgService(),
			bullionService:     getBullionService(),
			realtimeDatabase:   getFirebaseRealTimeDatabase(),
		}
		println("Trade User Service Initialized")
	}
	return TradeUserService
}

func (service *tradeUserServiceStruct) VerifyAndSendOtpForNewUser(tradeUser *bullion_main_server_interfaces.TradeUserBase, bullionId string) (*string, error) {
	users, err := service.tradeUserRepo.FindDuplicateUser(tradeUser.Email, tradeUser.Number, tradeUser.Email, bullionId)
	if err != nil {
		return nil, err
	}
	if len(*users) > 0 {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_DUPLICATE_USER,
			Message:    "User Exists With Matching With Wither Email,Number Or Username",
			Name:       "ERROR_DUPLICATE_USER",
		}
	}
	otpReqEntity, err := service.SendOtp(tradeUser.Name, tradeUser.Number, tradeUser.BullionId)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tokenString, err := service.accessTokenService.GenerateToken(&bullion_main_server_interfaces.GeneralPurposeTokenGeneration{
		GeneralPurposeTokenGeneration: &localJwt.GeneralPurposeTokenGeneration{
			RegisteredClaims: &jwt.RegisteredClaims{
				IssuedAt:  &jwt.NumericDate{Time: now},
				ExpiresAt: &jwt.NumericDate{Time: now.Add(time.Minute * 2)},
			},

			ExtraClaim: map[string]interface{}{
				"otpReqEntityId": otpReqEntity.ID,
				"tradeUser":      tradeUser,
			},
		},
		BullionId: bullionId,
	})
	if err != nil {
		return nil, err
	}
	return &tokenString, nil
}

func (service *tradeUserServiceStruct) verifyRegistrationToken(token string, returnTradeUser bool) (*bullion_main_server_interfaces.GeneralPurposeTokenGeneration, string, *bullion_main_server_interfaces.TradeUserBase, error) {
	claims, err := localJwt.VerifyToken[bullion_main_server_interfaces.GeneralPurposeTokenGeneration](service.accessTokenService, &token)
	if err != nil {
		return nil, "", nil, &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "OTP Req Token Expired",
			Name:       "ERROR_INVALID_INPUT",
			Extra:      err,
		}
	}
	otpReqId, ok := claims.ExtraClaim["otpReqEntityId"].(string)
	if !ok {
		return nil, "", nil, &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "OTP Req Id Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
	}
	if !returnTradeUser {
		return claims, otpReqId, nil, nil
	}
	tradeUserMap, ok := claims.ExtraClaim["tradeUser"]
	if !ok {
		return nil, "", nil, &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "TradeUser Details Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
	}
	tradeUser := new(bullion_main_server_interfaces.TradeUserBase)
	err = mapstructure.Decode(tradeUserMap, &tradeUser)
	if err != nil {
		return nil, "", nil, &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "TradeUser Details Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
	}
	return claims, otpReqId, tradeUser, nil
}

func (service *tradeUserServiceStruct) VerifyTokenAndResendOTP(token string) (*string, error) {
	claim, otpReqId, _, err := service.verifyRegistrationToken(token, false)
	if err != nil {
		return nil, err
	}
	err = service.sendMsgService.ResendOtp(otpReqId)
	if err != nil {
		return nil, err
	}
	claim.ExpiresAt = &jwt.NumericDate{Time: time.Now().Add(time.Minute * 2)}
	tokenString, err := service.accessTokenService.GenerateToken(claim)
	if err != nil {
		return nil, err
	}
	return &tokenString, nil
}

func (service *tradeUserServiceStruct) SendOtp(name string, number string, bullionId string) (*bullion_main_server_interfaces.OTPReqEntity, error) {
	bullionDetails, err := service.bullionService.GetBullionDetailsByBullionId(bullionId)
	if err != nil {
		return nil, err
	}
	entity, err := service.sendMsgService.SendOtp(&bullion_main_server_interfaces.OTPReqBase{
		BullionId: bullionId,
		Number:    number,
		Name:      name,
		Attempt:   0,
		ExpiresAt: time.Now(),
	}, &bullion_main_server_interfaces.MsgVariablesOTPReqStruct{
		BullionName: bullionDetails.Name,
		Name:        name,
		Number:      number,
	}, bullionDetails.BullionConfigs.OTPLength)

	if err != nil {
		return entity, err
	}
	return entity, nil
}

func (service *tradeUserServiceStruct) VerifyTokenAndVerifyOTP(token string, otp string) (*bullion_main_server_interfaces.TradeUserEntity, error) {
	_, otpReqId, tradeUser, err := service.verifyRegistrationToken(token, true)
	if err != nil {
		return nil, err
	}
	err = service.sendMsgService.VerifyOtp(otpReqId, otp)
	if err != nil {
		return nil, err
	}
	tradeUserEntity, err := service.RegisterNewTradeUser(tradeUser, &bullion_main_server_interfaces.TradeUserAdvanced{
		UserName: tradeUser.Name,
		IsActive: false,
		UNumber:  "0",
	}, &bullion_main_server_interfaces.TradeUserMargins{
		AllotedMargins: &bullion_main_server_interfaces.UserMarginsDataStruct{
			Gold:   0,
			Silver: 0,
		},
		UsedMargins: &bullion_main_server_interfaces.UserMarginsDataStruct{
			Gold:   0,
			Silver: 0,
		},
	})
	if err != nil {
		return nil, err
	}
	return tradeUserEntity, nil
}

func (service *tradeUserServiceStruct) RegisterNewTradeUser(base *bullion_main_server_interfaces.TradeUserBase, advance *bullion_main_server_interfaces.TradeUserAdvanced, margins *bullion_main_server_interfaces.TradeUserMargins) (*bullion_main_server_interfaces.TradeUserEntity, error) {
	entity := &bullion_main_server_interfaces.TradeUserEntity{
		TradeUserBase:     base,
		TradeUserAdvanced: advance,
		TradeUserMargins:  margins,
		BaseEntity:        &bullion_main_server_interfaces.BaseEntity{},
	}
	entity.CreateNew().UpdateUser()
	newUserNumber := 0
	service.firebaseDb.GetData("tradeUsersNumbers", []string{entity.BullionId}, &newUserNumber)
	newUserNumber++
	entity.UNumber = strconv.Itoa(newUserNumber)
	if bullionDetails, _ := service.bullionService.GetBullionDetailsByBullionId(entity.BullionId); bullionDetails != nil {
		if bullionDetails.BullionConfigs.DefaultGroupIdForTradeUser != "" {
			entity.GroupId = bullionDetails.BullionConfigs.DefaultGroupIdForTradeUser
		}
	}
	// raw, _ := bson.Marshal(entity)
	// fmt.Printf("raw: %v\n", string(raw))
	if err := utility_functions.ValidateReqInput(entity); err != nil {
		return nil, err
	}
	if _, err := service.tradeUserRepo.Save(entity); err != nil {
		return nil, err
	}
	service.firebaseDb.setPrivateData("tradeUsersNumbers", []string{entity.BullionId}, newUserNumber)
	go service.afterSuccessFullRegistration(entity.ID)
	return entity, nil
}

func (service *tradeUserServiceStruct) afterSuccessFullRegistration(userId string) {
	tradeUser, err := service.tradeUserRepo.FindOne(userId)
	if err != nil {
		return
	}
	bullionDetails, err := service.bullionService.GetBullionDetailsByBullionId(tradeUser.BullionId)
	if err != nil {
		return
	}
	service.eventBus.Publish(bullion_main_server_events.CreateTradeUserRegisteredEvent(tradeUser.BullionId, tradeUser, tradeUser.ID))
	service.sendMsgService.SendMessage(tradeUser.BullionId, "tradeUserRegistration", tradeUser.Number, &bullion_main_server_interfaces.MsgVariableTradeUserRegisteredSuccessFullyStruct{
		UserIdNumber: tradeUser.UNumber,
		BullionName:  bullionDetails.Name,
		Name:         tradeUser.Name,
		Number:       tradeUser.Number,
	})
}

func (service *tradeUserServiceStruct) LoginWithEmailAndPassword(email string, password string, bullionId string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	tradeUser, err := service.tradeUserRepo.FindOneByEmail(bullionId, email)
	if err != nil || tradeUser == nil {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusUnauthorized,
			Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
			Message:    "User Not Registered With Number",
			Name:       "Number not found",
		}
	}
	return service.generateTokensForTradeUserWithPasswordMatching(tradeUser, password)
}
func (service *tradeUserServiceStruct) LoginWithUNumberAndPassword(uNumber string, password string, bullionId string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	tradeUser, err := service.tradeUserRepo.FindOneByUNumber(bullionId, uNumber)
	if err != nil || tradeUser == nil {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusUnauthorized,
			Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
			Message:    "User Not Registered With Number",
			Name:       "Number not found",
		}
	}
	return service.generateTokensForTradeUserWithPasswordMatching(tradeUser, password)
}
func (service *tradeUserServiceStruct) LoginWithNumberAndPassword(number string, password string, bullionId string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	tradeUser, err := service.tradeUserRepo.FindOneByNumber(bullionId, number)
	if err != nil || tradeUser == nil {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusUnauthorized,
			Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
			Message:    "User Not Registered With Number",
			Name:       "Number not found",
		}
	}
	return service.generateTokensForTradeUserWithPasswordMatching(tradeUser, password)
}

func (service *tradeUserServiceStruct) GetTradeUserById(id string) (*bullion_main_server_interfaces.TradeUserEntity, error) {
	return service.tradeUserRepo.FindOne(id)
}

func (service *tradeUserServiceStruct) UpdateTradeUser(entity *bullion_main_server_interfaces.TradeUserEntity, adminId string) error {
	user, err := service.tradeUserRepo.FindOne(entity.ID)
	if err != nil {
		return err
	}
	if user.BullionId != entity.BullionId {
		return &interfaces.RequestError{
			StatusCode: http.StatusUnauthorized,
			Code:       interfaces.ERROR_PERMISSION_NOT_ALLOWED,
			Message:    "Cannot Update Different Bullion Id User",
			Name:       "CANNOT_UPDATE_DIFFERENT_BULLION_ID_USER",
		}
	}
	user.TradeUserBase = entity.TradeUserBase
	user.TradeUserAdvanced.IsActive = entity.TradeUserAdvanced.IsActive
	user.TradeUserMargins = entity.TradeUserMargins
	// TODO: Password Entity
	// user.Password = entity.Password
	if _, err := service.tradeUserRepo.Save(user); err != nil {
		return err
	}
	service.eventBus.Publish(bullion_main_server_events.CreateTradeUserUpdated(entity.BullionId, user, adminId))
	return nil
}

func (service *tradeUserServiceStruct) generateTokensForTradeUserWithPasswordMatching(tradeUser *bullion_main_server_interfaces.TradeUserEntity, password string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	if tradeUser.TradeUserBase.RawPassword != password {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusUnauthorized,
			Code:       interfaces.ERROR_INVALID_PASSWORD,
			Message:    "Incorrect Password",
			Name:       "ERROR_INVALID_PASSWORD",
		}
	}
	if !tradeUser.IsActive {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusUnauthorized,
			Code:       interfaces.ERROR_PERMISSION_NOT_ALLOWED,
			Message:    "Account Is Not Active Please Contact Admin",
			Name:       "ERROR_PERMISSION_NOT_ALLOWED",
		}
	}
	return service.generateTokensForTradeUser(tradeUser)
}

func (service *tradeUserServiceStruct) FindAndReturnAllInActiveTradeUsers(bullionId string) (*[]bullion_main_server_interfaces.TradeUserEntity, error) {
	return service.tradeUserRepo.FindAllInActiveUser(bullionId)
}
func (service *tradeUserServiceStruct) FindOneUserById(id string) (*bullion_main_server_interfaces.TradeUserEntity, error) {
	return service.tradeUserRepo.FindOne(id)
}
func (service *tradeUserServiceStruct) TradeUserChangeStatus(id string, bullionId string, isActive bool, adminId string) error {
	entity, err := service.tradeUserRepo.FindOne(id)
	if entity.BullionId != bullionId {
		return &interfaces.RequestError{
			StatusCode: http.StatusUnauthorized,
			Code:       interfaces.ERROR_MISMATCH_BULLION_ID,
			Message:    "Bullion Id Mismatch For Trade User",
			Name:       "BULLION_ID_MISMATCH_FOR_TRADE_USER",
		}
	}

	if err != nil {
		return err
	}
	entity.IsActive = isActive

	if _, err := service.tradeUserRepo.Save(entity); err != nil {
		return err
	}
	if isActive {
		service.eventBus.Publish(bullion_main_server_events.CreateTradeUserActivatedEvent(entity.BullionId, entity, adminId))
	} else {
		service.eventBus.Publish(bullion_main_server_events.CreateTradeUserDisabledEvent(entity.BullionId, entity, adminId))

	}
	return nil
}

// func (service *tradeUserServiceStruct) FindUserByNumberAndPassword(number string, password string, bullionId string) (*bullion_main_server_interfaces.TradeUserEntity, error) {
// 	tradeUser, err := service.tradeUserRepo.FindOneByNumber(bullionId, number)
// 	if err != nil || tradeUser == nil {
// 		return nil, &interfaces.RequestError{
// 			StatusCode: http.StatusUnauthorized,
// 			Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
// 			Message:    "User Not Registered With Number",
// 			Name:       "Number not found",
// 		}
// 	}
// 	if !tradeUser.MatchPassword(password) {
// 		return nil, &interfaces.RequestError{
// 			StatusCode: http.StatusUnauthorized,
// 			Code:       interfaces.ERROR_INVALID_PASSWORD,
// 			Message:    "Incorrect Password",
// 			Name:       "ERROR_INVALID_PASSWORD",
// 		}
// 	}
// 	return tradeUser, nil
// }

//	func (service *tradeUserServiceStruct) generateTokensForTradeUserById(userId string) (*bullion_main_server_interfaces.TokenResponseBody, error) {
//		user, err := service.tradeUserRepo.FindOne(userId)
//		if err != nil {
//			return nil, err
//		}
//		return service.generateTokensForTradeUser(user)
//	}
func (service *tradeUserServiceStruct) generateTokensForTradeUser(user *bullion_main_server_interfaces.TradeUserEntity) (*bullion_main_server_interfaces.TokenResponseBody, error) {
	return generateTokens(user.ID, user.BullionId, bullion_main_server_interfaces.ROLE_TRADE_USER)
}

// func (service *tradeUserServiceStruct) UpdateTradeUserDetails(){}

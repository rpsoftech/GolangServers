package bullion_main_server_services

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_events "github.com/rpsoftech/golang-servers/servers/bullion/main-server/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

type sendMsgService struct {
	redisRepo                *redis.RedisClientStruct
	eventBus                 *eventBusService
	firebaseDb               *firebaseDatabaseService
	bullionService           *bullionDetailsService
	regExpForMessageVariable *regexp.Regexp
}

var SendMsgService *sendMsgService

func getSendMsgService() *sendMsgService {
	if SendMsgService == nil {
		SendMsgService = &sendMsgService{
			redisRepo:                redis.InitRedisAndRedisClient(),
			eventBus:                 getEventBusService(),
			firebaseDb:               getFirebaseRealTimeDatabase(),
			bullionService:           getBullionService(),
			regExpForMessageVariable: regexp.MustCompile(`##\S*##`),
		}
		println("Send Msg Service Initialized")
	}
	return SendMsgService
}

func (s *sendMsgService) SendOtp(otpReq *bullion_main_server_interfaces.OTPReqBase, variable *bullion_main_server_interfaces.MsgVariablesOTPReqStruct, otpLength int) (*bullion_main_server_interfaces.OTPReqEntity, error) {
	data := s.redisRepo.GetStringData("otp/" + otpReq.BullionId + "/" + otpReq.Number)
	if len(data) > 0 {
		return nil, &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_OTP_ALREADY_SENT,
			Message:    "Otp Already Sent",
			Name:       "ERROR_OTP_ALREADY_SENT",
		}
	}
	variable.OTP = GenerateOTP(otpLength)
	entity := bullion_main_server_interfaces.CreateOTPEntity(otpReq, variable.OTP)
	err := s.prepareAndSendOTP(entity, variable)
	if err != nil {
		return nil, err
	}
	s.eventBus.Publish(bullion_main_server_events.CreateOtpSentEvent(entity))
	return entity, nil
}

func (s *sendMsgService) ResendOtp(otpReqId string) error {
	data := s.redisRepo.GetStringData("otp/" + otpReqId)
	if data == "" {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_OTP_EXPIRED,
			Message:    "OTP Req Expired",
			Name:       "ERROR_OTP_EXPIRED",
		}
	}
	otpReqEntity := new(bullion_main_server_interfaces.OTPReqEntity)
	err := json.Unmarshal([]byte(data), otpReqEntity)
	if err != nil {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusInternalServerError,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Unable to parse OTP REQ JSON",
		}
	}
	otpReqEntity.RestoreTimeStamp()
	if time.Now().Before(otpReqEntity.ModifiedAt.Add(time.Second * 15)) {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_OTP_ALREADY_SENT,
			Message:    "Please Wait For 15 Seconds Before Requesting",
			Name:       "ERROR_OTP_ALREADY_SENT",
		}
	}
	if otpReqEntity.Attempt >= 5 {

		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_TOO_MANY_ATTEMPTS,
			Message:    "OTP REQUESTED TOO MANY TIMES, Wait for 2 Minutes Before Requesting Again",
			Name:       "ERROR_TOO_MANY_ATTEMPTS",
		}
	}
	otpReqEntity.NewAttempt()
	bullionDetails, err := s.bullionService.GetBullionDetailsByBullionId(otpReqEntity.BullionId)
	if err != nil {
		return err
	}
	err = s.prepareAndSendOTP(otpReqEntity, &bullion_main_server_interfaces.MsgVariablesOTPReqStruct{
		OTP:         otpReqEntity.OTP,
		Name:        otpReqEntity.Name,
		Number:      otpReqEntity.Number,
		BullionName: bullionDetails.Name,
	})
	if err != nil {
		return err
	}
	s.eventBus.Publish(bullion_main_server_events.CreateOtpResendEvent(otpReqEntity))
	return err
}

func (s *sendMsgService) VerifyOtp(otpReqId string, otp string) error {
	data := s.redisRepo.GetStringData("otp/" + otpReqId)
	if data == "" {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_OTP_EXPIRED,
			Message:    "OTP Req Expired",
			Name:       "ERROR_OTP_EXPIRED",
		}
	}
	otpReqEntity := new(bullion_main_server_interfaces.OTPReqEntity)
	err := json.Unmarshal([]byte(data), otpReqEntity)
	if err != nil {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusInternalServerError,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Unable to parse OTP REQ JSON",
		}
	}
	otpReqEntity.RestoreTimeStamp()
	if otpReqEntity.OTP != otp {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_OTP_INVALID,
			Message:    "Invalid OTP",
			Name:       "ERROR_OTP_INVALID",
		}
	}
	s.eventBus.Publish(bullion_main_server_events.CreateOtpVerifiedEvent(otpReqEntity))
	return nil
}

func (s *sendMsgService) prepareAndSendOTP(otpReq *bullion_main_server_interfaces.OTPReqEntity, variable *bullion_main_server_interfaces.MsgVariablesOTPReqStruct) error {
	msgTemplate := new(bullion_main_server_interfaces.MsgTemplateBase)
	err := s.firebaseDb.GetData("msgTemplates", []string{otpReq.BullionId, "otp"}, msgTemplate)
	if msgTemplate.WhatsappTemplate == "" && msgTemplate.MSG91Id == "" {
		// TODO Throw Critical Error Which needs to be reported
		println("Something Went Wrong While Fetching OTP Templates")
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusInternalServerError,
			Code:       interfaces.ERROR_WHILE_FETCHING_MESSAGE_TEMPLATE,
			Message:    "OTP Template Error",
			Name:       "ERROR_WHILE_FETCHING_MESSAGE_TEMPLATE",
			Extra:      msgTemplate,
		}
	}
	if err != nil {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusInternalServerError,
			Code:       interfaces.ERROR_WHILE_FETCHING_MESSAGE_TEMPLATE,
			Message:    "OTP Template NOT Found",
			Name:       "ERROR_WHILE_FETCHING_MESSAGE_TEMPLATE",
			Extra:      err,
		}
	}
	err = s.saveAndUpdateOTPService(otpReq)
	if err != nil {
		return err
	}
	err = s.sendWhatsappMessage(msgTemplate.WhatsappTemplate, "OTP", variable, &bullion_main_server_interfaces.MsgEntity{
		BullionId:  otpReq.BullionId,
		Number:     otpReq.Number,
		BaseEntity: otpReq.BaseEntity,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *sendMsgService) saveAndUpdateOTPService(otpEntity *bullion_main_server_interfaces.OTPReqEntity) error {
	otpEntity.ExpiresAt = otpEntity.ExpiresAt.Add(120 * time.Second)
	otpEntity.AddTimeStamps()
	otpEntityStringBytes, err := json.Marshal(otpEntity)
	if err != nil {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusInternalServerError,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Unable convert OTP REQ to string",
			Name:       "OTPReq Entity Marshal Error",
			Extra:      err,
		}
	}
	otpEntityString := string(otpEntityStringBytes)
	s.redisRepo.SetStringData(fmt.Sprintf("otp/%s/%s", otpEntity.BullionId, otpEntity.Number), otpEntityString, 120)
	s.redisRepo.SetStringData(fmt.Sprintf("otp/%s", otpEntity.ID), otpEntityString, 120)
	return nil
}

func (s *sendMsgService) SendMessage(bullionId string, templateName string, number string, variables interface{}) error {
	msgTemplate := new(bullion_main_server_interfaces.MsgTemplateBase)
	err := s.firebaseDb.GetData("msgTemplates", []string{bullionId, templateName}, msgTemplate)
	if err != nil {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusInternalServerError,
			Code:       interfaces.ERROR_WHILE_FETCHING_MESSAGE_TEMPLATE,
			Message:    "Message Template NOT Found",
			Name:       "ERROR_WHILE_FETCHING_MESSAGE_TEMPLATE",
			Extra:      err,
		}
	}
	msgEntity := &bullion_main_server_interfaces.MsgEntity{
		BullionId: bullionId,
		Number:    number,
	}
	msgEntity.Create()
	if msgTemplate.WhatsappTemplate != "" {
		s.sendWhatsappMessage(msgTemplate.WhatsappTemplate, templateName, variables, msgEntity)
	}
	return nil
}

func (s *sendMsgService) sendWhatsappMessage(template string, templateName string, variables interface{}, msgEntity *bullion_main_server_interfaces.MsgEntity) error {
	jsonMap, err := utility_functions.StructToStringMap(variables)
	if err != nil {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusInternalServerError,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Error While converting Struct To JSON",
			Name:       "CONVERSATION_ERROR",
			Extra:      err,
		}
	}
	routeToPost := msgEntity.BullionId
	bullionDetails, err := s.bullionService.GetBullionDetailsByBullionId(msgEntity.BullionId)
	if err != nil {
		return err
	}
	if !bullionDetails.BullionConfigs.HaveCustomWhatsappAgent {
		routeToPost = "common"
	}
	message := s.processMessage(template, &jsonMap)
	err = s.firebaseDb.setPrivateData("whatsappMessage", []string{routeToPost, msgEntity.ID}, map[string]string{
		"message": message,
		"number":  msgEntity.Number,
	})
	if err != nil {
		return &bullion_main_server_interfaces.RequestError{
			StatusCode: http.StatusInternalServerError,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Error Posting Whatsapp Message to Firebase",
			Name:       "CONVERSATION_ERROR",
			Extra:      err,
		}
	}
	s.eventBus.Publish(bullion_main_server_events.CreateWhatsappMessageSendEvent(msgEntity.BullionId, templateName, msgEntity.Number, message))
	return nil
}
func (s *sendMsgService) processMessage(template string, variables *map[string]string) string {
	for key, value := range *variables {
		template = strings.ReplaceAll(template, fmt.Sprintf("##%s##", key), value)
	}
	template = s.regExpForMessageVariable.ReplaceAllString(template, "")
	return template
}

var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

func GenerateOTP(max int) string {
	b := make([]byte, max)
	n, err := io.ReadAtLeast(rand.Reader, b, max)
	if n != max {
		panic(err)
	}
	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b)
}

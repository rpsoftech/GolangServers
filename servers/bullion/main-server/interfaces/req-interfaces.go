package bullion_main_server_interfaces

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/validator"
)

const (
	REQ_LOCAL_KEY_ROLE           = "UserRole"
	REQ_LOCAL_ERROR_KEY          = "Error"
	REQ_LOCAL_UserID             = "UserId"
	REQ_LOCAL_BullionId_KEY      = "BullionId"
	REQ_LOCAL_KEY_TOKEN_RAW_DATA = "TokenRawData"
)

type RequestError struct {
	StatusCode    int    `json:"-"`
	Code          int    `json:"code"`
	Message       string `json:"message"`
	Name          string `json:"name"`
	Extra         any    `json:"extra,omitempty"`
	LogTheDetails bool   `json:"-"`
}

func (r *RequestError) Error() string {
	return fmt.Sprintf("status %d: err %v", r.StatusCode, r.Message)
}
func (r *RequestError) AppendValidationErrors(errs []validator.ErrorResponse) *RequestError {
	// return fmt.Sprintf("status %d: err %v", r.StatusCode, r.Message)
	for index, element := range errs {
		if index != 0 {
			r.Message += "\n"
		}
		r.Message += fmt.Sprintf("FieldName:- %s,Passed Value:- %s,Failed Tag:- %s", element.FailedField, element.Value, element.Tag)
	}
	return r
}

func ValidateBullionIdMatchingInToken(c *fiber.Ctx, bullionId string) error {
	id, ok := c.Locals(REQ_LOCAL_BullionId_KEY).(string)
	if !ok || bullionId != id {
		return &RequestError{
			StatusCode: http.StatusForbidden,
			Code:       interfaces.ERROR_MISMATCH_BULLION_ID,
			Message:    "Your can not access this resource due to different bullionId",
			Name:       "ERROR_MISMATCH_BULLION_ID",
			Extra:      fmt.Sprintf("Expected := %s Got := %s", id, bullionId),
		}
	}
	return nil
}
func ExtractTokenUserIdFromCtx(c *fiber.Ctx) (string, error) {
	id, ok := c.Locals(REQ_LOCAL_UserID).(string)
	if !ok {
		return "", &RequestError{
			StatusCode: http.StatusForbidden,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Your can not access this resource due to different UserId",
			Name:       "ERROR_USER_ID_NOT_FOUND",
		}
	}
	return id, nil
}

func ExtractBullionIdFromCtx(c *fiber.Ctx) (string, error) {
	id, ok := c.Locals(REQ_LOCAL_BullionId_KEY).(string)
	if !ok {
		return "", &RequestError{
			StatusCode: http.StatusForbidden,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Your can not access this resource due to different UserId",
			Name:       "ERROR_BULLION_ID_NOT_FOUND",
		}
	}
	return id, nil
}

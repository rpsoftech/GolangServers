package soham_whatsapp_server_middleware

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

func TestCheckClientToken(t *testing.T) {
	base := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	tokenFor := func(n string) string { return utility_functions.UUIDv5(base, n) }

	tests := []struct {
		name     string
		number   string
		token    string
		wantCode int // 0 means no error
	}{
		{"valid", "919876543210", tokenFor("919876543210"), 0},
		{"wrong token", "919876543210", tokenFor("919876543211"), soham_common_req_keys.ERROR_GENERATE_TOKEN_MISMATCH},
		{"token with different length", "7", tokenFor("7")[:10], soham_common_req_keys.ERROR_GENERATE_TOKEN_MISMATCH},
		{"leading zero", "007", tokenFor("007"), soham_common_req_keys.ERROR_NUMBER_TOKEN_NOT_PASSED},
		{"plus sign", "+7", tokenFor("+7"), soham_common_req_keys.ERROR_NUMBER_TOKEN_NOT_PASSED},
		{"zero", "0", tokenFor("0"), soham_common_req_keys.ERROR_NUMBER_TOKEN_NOT_PASSED},
		{"negative", "-7", tokenFor("-7"), soham_common_req_keys.ERROR_NUMBER_TOKEN_NOT_PASSED},
		{"not a number", "abc", tokenFor("abc"), soham_common_req_keys.ERROR_NUMBER_TOKEN_NOT_PASSED},
		{"space", " 7", tokenFor(" 7"), soham_common_req_keys.ERROR_NUMBER_TOKEN_NOT_PASSED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkClientToken(base, tt.number, tt.token)
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var reqErr *interfaces.RequestError
			if !errors.As(err, &reqErr) {
				t.Fatalf("want *RequestError, got %v", err)
			}
			if reqErr.Code != tt.wantCode || reqErr.StatusCode != 403 {
				t.Fatalf("got code %d status %d, want code %d status 403", reqErr.Code, reqErr.StatusCode, tt.wantCode)
			}
		})
	}
}

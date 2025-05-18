package bullion_main_server_interfaces

import (
	"testing"
)

func TestAdminUserEntity(t *testing.T) {
	e := &AdminUserEntity{}
	// // c.CreateNewId()
	// Admin()
	t.Run("Create New Admin Entity", func(t *testing.T) {
		if e.BaseEntity != nil {
			t.Fatalf("BaseEntity is not nil")
		}
		if e.UserRolesInterface != nil {
			t.Fatalf("UserRolesInterface is not nil")
		}
		e.CreateNewEntity("admin", "password", "nickName", "bullionId")
		if e.BaseEntity == nil {
			t.Fatalf("BaseEntity is nil")
		}
		if e.UserName != "admin" {
			t.Fatalf("UserName is not admin")
		}
		if e.Password != "password" {
			t.Fatalf("Password is not password")
		}
		if e.NickName != "nickName" {
			t.Fatalf("NickName is not nickName")
		}
		if e.BullionId != "bullionId" {
			t.Fatalf("BullionId is not bullionId")
		}
		if e.UserRolesInterface == nil {
			t.Fatalf("UserRolesInterface is nil")
		}
		if e.UserRolesInterface.Role != ROLE_ADMIN {
			t.Fatalf("UserRolesInterface Role is not ROLE_ADMIN")
		}
		testBaseEntityCreateNewId(t, e.BaseEntity)
		t.Run("password match", func(t *testing.T) {
			if !e.MatchPassword("password") {
				t.Fatalf("Password does not match")
			}
		})
	})
}

package bullion_main_server_interfaces

import (
	"testing"
	"time"
)

func testBaseEntityCreateNewId(t *testing.T, c *BaseEntity) {

	if c.ID == "" {
		t.Fatalf("Id is empty")
	}

	if c.CreatedAt.IsZero() {
		t.Fatalf("CreatedAt is empty")
	}
	if c.ModifiedAt.IsZero() {
		t.Fatalf("ModifiedAt is empty")
	}
	if !c.CreatedAtExported.IsZero() {
		t.Fatalf("CreatedAtExported is not empty %d", c.CreatedAtExported.Unix())
	}
	if !c.ModifiedAtExported.IsZero() {
		t.Fatalf("ModifiedAtExported is not empty")
	}
}

func TestBaseEntity(t *testing.T) {
	c := &BaseEntity{}
	c.createNewId()
	t.Run("Create New ID", func(t *testing.T) {
		id := c.ID
		c.createNewId()
		if c.ID == id {
			t.Fatalf("Id Should Be Different")
		}
		testBaseEntityCreateNewId(t, c)
	})
	t.Run("AddTimeStamps", func(t *testing.T) {
		now := time.Now()
		c.CreatedAt = now
		c.ModifiedAt = now
		c.AddTimeStamps()
		if c.CreatedAtExported != now {
			t.Fatalf("CreatedAt and CreatedAtExported Should be same")
		}
		if c.ModifiedAtExported != now {
			t.Fatalf("ModifiedAt and ModifiedAtExported Should be same")
		}
	})
	t.Run("RestoreTimeStamp", func(t *testing.T) {
		now := time.Now()
		c.CreatedAtExported = now
		c.ModifiedAtExported = now
		c.RestoreTimeStamp()
		if c.CreatedAt != now {
			t.Fatalf("CreatedAt and CreatedAtExported Should be same")
		}
		if c.ModifiedAt != now {
			t.Fatalf("ModifiedAt and ModifiedAtExported Should be same")
		}
	})
	t.Run("Updated", func(t *testing.T) {
		now := time.Now()
		c.ModifiedAt = now
		c.Updated()
		if c.ModifiedAt.Before(now) {
			t.Fatalf("ModifiedAt Should be greater than now")
		}
	})
}

package bullion_main_server_interfaces

import "golang.org/x/crypto/bcrypt"

type passwordEntity struct {
	Password []byte `bson:"password" json:"-" validate:"required"`
	// Password []byte `bson:"hashedPassword" json:"-" validate:"required"`
}

func (e *passwordEntity) MatchPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(e.Password), []byte(password))
	if err != nil {
		return false
	} else {
		return true
	}

}

func CreatePasswordEntity(password string) *passwordEntity {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return &passwordEntity{
			Password: []byte(password),
		}
	}
	return &passwordEntity{
		Password: hashedPassword,
	}
}

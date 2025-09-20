package func_management_interfaces

import (
	"time"

	"github.com/rpsoftech/golang-servers/interfaces"
)

// {
//     MainEvent
//     name
//     description
//     function-dates
// }

type MainEventEntity struct {
	*interfaces.BaseEntity `bson:"inline"`
	Name                   string      `bson:"name" json:"name"`
	Description            string      `bson:"description" json:"description"`
	FunctionDates          []time.Time `bson:"functionDates" json:"functionDates"`
}

func CreateMainEvent(name string, description string, functionDates []time.Time) *MainEventEntity {
	main := &MainEventEntity{
		BaseEntity:    &interfaces.BaseEntity{},
		Name:          name,
		Description:   description,
		FunctionDates: functionDates,
	}
	main.CreateNewId()
	return main
}

package bullion_main_server_services

import (
	bullion_main_server_events "github.com/rpsoftech/golang-servers/servers/bullion/main-server/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type FlagServiceStruct struct {
	firebaseDb *firebaseDatabaseService
	eventBus   *eventBusService
	flagsMap   map[string]*bullion_main_server_interfaces.FlagsInterface
}

var FlagService *FlagServiceStruct

func init() {
	getFlagService()
}

func getFlagService() *FlagServiceStruct {
	if FlagService == nil {
		FlagService = &FlagServiceStruct{
			firebaseDb: getFirebaseRealTimeDatabase(),
			eventBus:   getEventBusService(),
			flagsMap:   make(map[string]*bullion_main_server_interfaces.FlagsInterface),
		}
		println("Flag Service Initialized")
	}
	return FlagService
}

func (s *FlagServiceStruct) UpdateFlags(entity *bullion_main_server_interfaces.FlagsInterface, adminId string) (*bullion_main_server_interfaces.FlagsInterface, error) {
	if err := s.firebaseDb.SetPublicData(entity.BullionId, []string{"flags"}, entity); err != nil {
		return nil, err
	}
	s.eventBus.Publish(bullion_main_server_events.FlagsUpdatedEvent(entity, adminId))
	s.flagsMap[entity.BullionId] = entity
	return entity, nil
}

func (s *FlagServiceStruct) GetFlags(bullionId string) (*bullion_main_server_interfaces.FlagsInterface, error) {
	if entity := s.flagsMap[bullionId]; entity != nil {
		return entity, nil
	}

	entity := new(bullion_main_server_interfaces.FlagsInterface)
	if err := s.firebaseDb.GetPublicData(bullionId, []string{"flags"}, entity); err != nil {
		return nil, err
	}
	s.flagsMap[bullionId] = entity
	return entity, nil
}

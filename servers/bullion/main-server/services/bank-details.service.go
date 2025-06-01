package bullion_main_server_services

import (
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_events "github.com/rpsoftech/golang-servers/servers/bullion/main-server/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_repos "github.com/rpsoftech/golang-servers/servers/bullion/main-server/repos"
)

type bankDetailsService struct {
	bankDetailsRepo *bullion_main_server_repos.BankDetailsRepoStruct
	eventBus        *eventBusService
}

var BankDetailsService *bankDetailsService

func init() {
	BankDetailsService = &bankDetailsService{
		eventBus:        getEventBusService(),
		bankDetailsRepo: bullion_main_server_repos.BankDetailsRepo,
	}
	println("Bank Details Service Initialized")
}

func (s *bankDetailsService) GetBankDetailsByBullionId(id string) (*[]bullion_main_server_interfaces.BankDetailsEntity, error) {
	return s.bankDetailsRepo.GetAllByBullionId(id)
}

func (s *bankDetailsService) addUpdateBankDetails(entity *bullion_main_server_interfaces.BankDetailsEntity) (*bullion_main_server_interfaces.BankDetailsEntity, error) {
	_, err := s.bankDetailsRepo.Save(entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}
func (s *bankDetailsService) UpdateBankDetails(entity *bullion_main_server_interfaces.BankDetailsBase, id string, adminId string) (*bullion_main_server_interfaces.BankDetailsEntity, error) {
	entityFromDb, err := s.bankDetailsRepo.FindOne(id)
	if err != nil {
		return nil, err
	}
	if entityFromDb.BullionId != entity.BullionId {
		return nil, &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_MISMATCH_BULLION_ID,
			Message:    "You do not have access to this Bank Details",
			Name:       "ERROR_MISMATCH_BULLION_ID",
		}
	}
	entityFromDb.BankDetailsBase = entity
	_, err = s.addUpdateBankDetails(entityFromDb)
	if err != nil {
		return nil, err
	}
	event := bullion_main_server_events.CreateBankDetailsUpdatedEvent(entityFromDb, adminId)
	s.eventBus.Publish(event)
	return entityFromDb, err
}
func (s *bankDetailsService) AddNewBankDetails(base *bullion_main_server_interfaces.BankDetailsBase, adminId string) (*bullion_main_server_interfaces.BankDetailsEntity, error) {
	entity := bullion_main_server_interfaces.CreateNewBankDetails(base)
	_, err := s.addUpdateBankDetails(entity)
	if err != nil {
		return nil, err
	}
	event := bullion_main_server_events.CreateNewBankDetailsCreated(entity, adminId)
	s.eventBus.Publish(event)
	return entity, err
}
func (s *bankDetailsService) DeleteBankDetails(entity *bullion_main_server_interfaces.BankDetailsBase, id string, adminId string) (*bullion_main_server_interfaces.BankDetailsEntity, error) {
	entityFromDb, err := s.bankDetailsRepo.FindOne(id)
	if err != nil {
		return nil, err
	}
	if entityFromDb.BullionId != entity.BullionId {
		return nil, &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_MISMATCH_BULLION_ID,
			Message:    "You do not have access to this Bank Details",
			Name:       "ERROR_MISMATCH_BULLION_ID",
		}
	}
	err = s.bankDetailsRepo.DeleteById(id)
	if err != nil {
		return nil, err
	}
	event := bullion_main_server_events.CreateBankDetailsDeletedEvent(entity, id, adminId)
	s.eventBus.Publish(event)
	return entityFromDb, err
}

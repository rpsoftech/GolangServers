package bullion_main_server_services

import (
	bullion_main_server_events "github.com/rpsoftech/golang-servers/servers/bullion/main-server/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_repos "github.com/rpsoftech/golang-servers/servers/bullion/main-server/repos"
)

type bankRateService struct {
	bankRateRepo            *bullion_main_server_repos.BankRateCalcRepoStruct
	eventBus                *eventBusService
	firebaseDatabaseService *firebaseDatabaseService
}

var BankRateCalcService *bankRateService

func init() {
	getBankRateService()
}

func getBankRateService() *bankRateService {
	if BankRateCalcService == nil {
		BankRateCalcService = &bankRateService{
			eventBus:                getEventBusService(),
			bankRateRepo:            bullion_main_server_repos.BankRateCalcRepo,
			firebaseDatabaseService: getFirebaseRealTimeDatabase(),
		}
		println("Bank Rate Service Initialized")
	}
	return BankRateCalcService
}

func (service *bankRateService) GetBankRateCalcByBullionId(bullionId string) (*bullion_main_server_interfaces.BankRateCalcEntity, error) {
	return service.bankRateRepo.FindOneByBullionId(bullionId)
}

func (service *bankRateService) SaveBankRateCalc(gold *bullion_main_server_interfaces.BankRateCalcBase, silver *bullion_main_server_interfaces.BankRateCalcBase, bullionId string, adminId string) (*bullion_main_server_interfaces.BankRateCalcEntity, error) {
	entity, err := service.GetBankRateCalcByBullionId(bullionId)
	if err != nil {
		entity = &bullion_main_server_interfaces.BankRateCalcEntity{
			BullionId:   bullionId,
			GOLD_SPOT:   gold,
			SILVER_SPOT: silver,
		}
		entity.CreateNewBankRateCalc()
	} else {
		entity.GOLD_SPOT = gold
		entity.SILVER_SPOT = silver
	}
	_, err = service.bankRateRepo.Save(entity)
	if err != nil {
		return nil, err
	}
	service.eventBus.Publish(bullion_main_server_events.BankRateCalcUpdatedEvent(entity, adminId))
	service.firebaseDatabaseService.SetPublicData(bullionId, []string{"bankRateCalc"}, entity)
	return entity, nil
}

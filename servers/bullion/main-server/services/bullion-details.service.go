package bullion_main_server_services

import (
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_repos "github.com/rpsoftech/golang-servers/servers/bullion/main-server/repos"
)

type bullionDetailsService struct {
	BullionSiteInfoRepo           *bullion_main_server_repos.BullionSiteInfoRepoStruct
	bullionSiteInfoMapById        map[string]*bullion_main_server_interfaces.BullionSiteInfoEntity
	bullionSiteInfoMapByShortName map[string]*bullion_main_server_interfaces.BullionSiteInfoEntity
}

var BullionDetailsService *bullionDetailsService

func init() {
	getBullionService()
}

func getBullionService() *bullionDetailsService {
	if BullionDetailsService == nil {
		BullionDetailsService = &bullionDetailsService{
			BullionSiteInfoRepo:           bullion_main_server_repos.BullionSiteInfoRepo,
			bullionSiteInfoMapById:        make(map[string]*bullion_main_server_interfaces.BullionSiteInfoEntity),
			bullionSiteInfoMapByShortName: make(map[string]*bullion_main_server_interfaces.BullionSiteInfoEntity),
		}
		println("Bullion Site Details Initialized")
	}
	return BullionDetailsService
}

func (service *bullionDetailsService) GetBullionDetailsByShortName(shortName string) (*bullion_main_server_interfaces.BullionSiteInfoEntity, error) {
	if bullion, ok := service.bullionSiteInfoMapByShortName[shortName]; ok {
		return bullion, nil
	}
	bullion, err := service.BullionSiteInfoRepo.FindByShortName(shortName)
	if err != nil {
		return nil, err
	}
	service.bullionSiteInfoMapById[bullion.ID] = bullion
	service.bullionSiteInfoMapByShortName[shortName] = bullion
	return bullion, nil
}
func (service *bullionDetailsService) GetBullionDetailsByBullionId(id string) (*bullion_main_server_interfaces.BullionSiteInfoEntity, error) {
	if bullion, ok := service.bullionSiteInfoMapById[id]; ok {
		return bullion, nil
	}
	bullion, err := service.BullionSiteInfoRepo.FindOne(id)
	if err != nil {
		return nil, err
	}
	service.bullionSiteInfoMapById[id] = bullion
	service.bullionSiteInfoMapByShortName[bullion.ShortName] = bullion
	return bullion, nil
}

func (service *bullionDetailsService) UpdateBullionSiteDetails(details *bullion_main_server_interfaces.BullionSiteInfoEntity) (*bullion_main_server_interfaces.BullionSiteInfoEntity, error) {
	service.bullionSiteInfoMapById[details.ID] = details
	service.bullionSiteInfoMapByShortName[details.ShortName] = details
	return service.BullionSiteInfoRepo.Save(details)
}

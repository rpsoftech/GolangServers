package bullion_main_server_interfaces

import (
	"net/http"
	"slices"
	"time"

	"github.com/rpsoftech/golang-servers/interfaces"
)

type (
	OrderEntity struct {
		*BaseEntity           `bson:"inline"`
		*OrderBase            `bson:"inline"`
		*LimitWatcherRequired `bson:"inline"`
		*AfterSuccessOrder    `bson:"inline,omitempty"`
		*Identity             `bson:"inline"`
		FromAdmin             *FromAdmin      `bson:"inline,omitempty"`
		DeliveryData          *[]DeliveryData `bson:"inline,omitempty"`
	}
	OrderBase struct {
		BullionId   string      `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		OrderType   OrderType   `bson:"orderType" json:"orderType" validate:"required,enum=OrderStatus"`
		OrderStatus OrderStatus `bson:"orderStatus" json:"orderStatus" validate:"required,enum=OrderStatus"`
		BuySell     BuySell     `bson:"buySell" json:"buySell" validate:"required,enum=BuySell"`
		ProductName string      `bson:"productName" json:"productName" validate:"required,min=3,max=100"`
	}
	LimitWatcherRequired struct {
		ProductId         string  `json:"productId" bson:"productId" validate:"required,uuid"`
		GroupId           string  `json:"groupId" bson:"groupId" validate:"required,uuid"`
		ProductGroupMapId string  `json:"productGroupMapId" bson:"productGroupMapId" validate:"required,uuid"`
		Volume            float64 `json:"volume" bson:"volume" validate:"required"`
		Weight            int     `json:"weight" bson:"weight" validate:"required"`
		Price             float64 `json:"price" bson:"price" validate:"required"`
	}

	FromAdmin struct {
		PlacedBy string `bson:"placedBy,omitempty" json:"placedBy,omitempty" validate:"uuid"`
		AutoRate bool   `bson:"autoRate,omitempty" json:"autoRate,omitempty" validate:"boolean"`
	}

	AfterSuccessOrder struct {
		CalcSnapShot   *CalcSnapshotStruct `json:"calcSnapShot" bson:"calcSnapShot" validate:"required"`
		CalcSnapShotId string              `json:"calcSnapShotId" bson:"calcSnapShotId" validate:"required,uuid"`
		PgmSnapShot    *GroupPremiumBase   `json:"pgmSnapShot" bson:"pgmSnapShot" validate:"required"`
		BankRateCalc   *BankRateCalcBase   `json:"bankRateCalc" bson:"bankRateCalc"`
		McxPrice       float64             `json:"mcxPrice" bson:"mcxPrice" validate:"required"`
		ExecutedOn     time.Time           `json:"executedOn" bson:"executedOn" validate:"required"`
		McxOrderJSON   interface{}         `json:"mcxOrder,omitempty" bson:"mcxOrder,omitempty"`
	}

	Identity struct {
		UserId      string `bson:"userId" json:"userId" validate:"required,uuid"`
		MachineId   string `bson:"machineId,omitempty" json:"machineId,omitempty" validate:"required"`
		MachineName string `bson:"machineName,omitempty" json:"machineName,omitempty" validate:"required,min=3,max=100"`
		IpAddress   string `bson:"ipAddress" json:"ipAddress"`
	}

	DeliveryData struct {
		Weight        int       `bson:"weight" json:"weight" validate:"required"`
		PendingWeight int       `bson:"pendingWeight" json:"pendingWeight" validate:"required"`
		DeliveredOn   time.Time `bson:"deliveredOn" json:"deliveredOn" validate:"required"`
	}
)

var AvailableForDelivery = &[]OrderStatus{OrderPlaced, LimitPassed, OrderPartialDelivered}

func (e *OrderEntity) DeliverWeight(weight int) (*OrderEntity, error) {
	if !slices.Contains(*AvailableForDelivery, e.OrderStatus) {
		return nil, &RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_ORDER_STATUS_FOR_DELIVERY,
			Message:    "Cannot deliver weight. Order status is not available for delivery",
			Name:       "OrderStatusNotAvailableForDelivery",
		}
	}
	pendingWeight := e.Weight
	if e.DeliveryData != nil {
		for _, dataD := range *e.DeliveryData {
			pendingWeight = pendingWeight - dataD.Weight
		}
	} else {
		e.DeliveryData = &[]DeliveryData{}
	}

	if weight > pendingWeight {
		return nil, &RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_WEIGHT_FOR_DELIVERY,
			Message:    "Cannot deliver weight. Weight is not available for delivery",
			Name:       "InvalidWeightForDelivery",
		}
	}

	*e.DeliveryData = append(*e.DeliveryData, DeliveryData{
		Weight:        weight,
		PendingWeight: pendingWeight - weight,
		DeliveredOn:   time.Now(),
	})
	pendingWeight = pendingWeight - weight
	e.OrderStatus = OrderPartialDelivered

	if pendingWeight == 0 {
		e.OrderStatus = OrderDelivered
	}

	return e, nil
}

func (e *OrderEntity) LimitPassedOrOrderPlaced(mcxPrice float64, calcSnapShot *CalcSnapshotStruct, calcSnapShotId string, pgmSnapShot *GroupPremiumBase, mcxOrderJSON interface{}) *OrderEntity {
	e.AfterSuccessOrder = &AfterSuccessOrder{
		CalcSnapShot:   calcSnapShot,
		CalcSnapShotId: calcSnapShotId,
		PgmSnapShot:    pgmSnapShot,
		McxPrice:       mcxPrice,
		ExecutedOn:     time.Now(),
		McxOrderJSON:   mcxOrderJSON,
	}
	return e
}

func CreateNewOrderEntity(base *OrderBase, limitWatch *LimitWatcherRequired, identity *Identity) *OrderEntity {
	b := &OrderEntity{
		BaseEntity:           &BaseEntity{},
		OrderBase:            base,
		LimitWatcherRequired: limitWatch,
		Identity:             identity,
	}
	b.createNewId()
	return b
}

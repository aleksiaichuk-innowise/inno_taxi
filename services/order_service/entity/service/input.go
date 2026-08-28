package service

type CreateOrderInput struct {
	UserID      string
	TaxiType    TaxiType
	Start       Location
	Destination Location
}

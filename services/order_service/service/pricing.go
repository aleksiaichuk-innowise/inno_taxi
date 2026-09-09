package service

import service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"

// Flat price-per-taxi-type placeholder, in minor currency units. The real
// Pricing Engine (distance/time/surge) is a separate, out-of-scope feature -
// see docs/superpowers/specs/2026-09-09-order-wallet-payment-design.md. This
// exists only to give the wallet charge/refund flow a non-zero amount.
const (
	priceEconomyMinorUnits  int64 = 500
	priceComfortMinorUnits  int64 = 800
	priceBusinessMinorUnits int64 = 1200
)

func priceForTaxiType(t service_dto.TaxiType) int64 {
	switch t {
	case service_dto.TaxiTypeEconomy:
		return priceEconomyMinorUnits
	case service_dto.TaxiTypeComfort:
		return priceComfortMinorUnits
	case service_dto.TaxiTypeBusiness:
		return priceBusinessMinorUnits
	default:
		return 0
	}
}

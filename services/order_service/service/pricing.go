package service

import (
	"math"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

// Base fare and per-km rate per taxi type, in minor currency units. Time and
// surge multipliers are out of scope: nothing in this codebase tracks route
// duration, traffic, or real-time demand, so there's no non-arbitrary input
// for them - see docs/superpowers/specs/2026-09-09-order-wallet-payment-design.md.
const (
	baseFareEconomyMinorUnits  int64 = 200
	baseFareComfortMinorUnits  int64 = 300
	baseFareBusinessMinorUnits int64 = 500

	perKmRateEconomyMinorUnits  int64 = 15
	perKmRateComfortMinorUnits  int64 = 25
	perKmRateBusinessMinorUnits int64 = 40
)

const earthRadiusKm = 6371.0

// haversineKm returns the great-circle distance between two points, in km.
func haversineKm(a, b service_dto.Location) float64 {
	aLat, aLng := a.Lat*math.Pi/180, a.Lng*math.Pi/180
	bLat, bLng := b.Lat*math.Pi/180, b.Lng*math.Pi/180

	dLat := bLat - aLat
	dLng := bLng - aLng

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(aLat)*math.Cos(bLat)*math.Sin(dLng/2)*math.Sin(dLng/2)

	return 2 * earthRadiusKm * math.Asin(math.Sqrt(h))
}

func priceForTrip(t service_dto.TaxiType, start, destination service_dto.Location) int64 {
	distanceKm := haversineKm(start, destination)

	switch t {
	case service_dto.TaxiTypeEconomy:
		return baseFareEconomyMinorUnits + int64(math.Round(float64(perKmRateEconomyMinorUnits)*distanceKm))
	case service_dto.TaxiTypeComfort:
		return baseFareComfortMinorUnits + int64(math.Round(float64(perKmRateComfortMinorUnits)*distanceKm))
	case service_dto.TaxiTypeBusiness:
		return baseFareBusinessMinorUnits + int64(math.Round(float64(perKmRateBusinessMinorUnits)*distanceKm))
	default:
		return 0
	}
}

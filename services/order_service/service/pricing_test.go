package service

import (
	"math"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

func TestHaversineKm_SamePoint(t *testing.T) {
	p := service_dto.Location{Lat: 51.5074, Lng: -0.1278}

	got := haversineKm(p, p)
	if got != 0 {
		t.Fatalf("expected 0 for identical points, got %v", got)
	}
}

func TestHaversineKm_KnownDistance(t *testing.T) {
	// London <-> Paris, well-known great-circle distance ~344km.
	london := service_dto.Location{Lat: 51.5074, Lng: -0.1278}
	paris := service_dto.Location{Lat: 48.8566, Lng: 2.3522}

	got := haversineKm(london, paris)
	want := 344.0
	if math.Abs(got-want) > 5 {
		t.Fatalf("got %v km, want ~%v km", got, want)
	}
}

func TestPriceForTrip_ZeroDistanceIsBaseFareOnly(t *testing.T) {
	p := service_dto.Location{Lat: 1, Lng: 1}

	got := priceForTrip(service_dto.TaxiTypeEconomy, p, p)
	if got != baseFareEconomyMinorUnits {
		t.Fatalf("got %d, want base fare %d", got, baseFareEconomyMinorUnits)
	}
}

func TestPriceForTrip_ScalesWithDistancePerTaxiType(t *testing.T) {
	start := service_dto.Location{Lat: 1, Lng: 1}
	dest := service_dto.Location{Lat: 2, Lng: 2}
	dist := haversineKm(start, dest)

	cases := []struct {
		taxiType service_dto.TaxiType
		baseFare int64
		perKm    int64
	}{
		{service_dto.TaxiTypeEconomy, baseFareEconomyMinorUnits, perKmRateEconomyMinorUnits},
		{service_dto.TaxiTypeComfort, baseFareComfortMinorUnits, perKmRateComfortMinorUnits},
		{service_dto.TaxiTypeBusiness, baseFareBusinessMinorUnits, perKmRateBusinessMinorUnits},
	}

	for _, c := range cases {
		want := c.baseFare + int64(math.Round(float64(c.perKm)*dist))
		got := priceForTrip(c.taxiType, start, dest)
		if got != want {
			t.Fatalf("%s: got %d, want %d", c.taxiType, got, want)
		}
	}
}

func TestPriceForTrip_UnknownTaxiTypeIsZero(t *testing.T) {
	start := service_dto.Location{Lat: 1, Lng: 1}
	dest := service_dto.Location{Lat: 2, Lng: 2}

	got := priceForTrip("unknown", start, dest)
	if got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

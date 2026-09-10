package analytic_service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnalyticGateway_GetDriverRatingStats_Found(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"average":4.5,"count":12}`))
	}))
	defer srv.Close()

	gw := NewAnalyticGateway(srv.URL, nil)
	avg, count, err := gw.GetDriverRatingStats(context.Background(), "driver-1")
	if err != nil {
		t.Fatalf("GetDriverRatingStats() error = %v, want nil", err)
	}
	if avg != 4.5 || count != 12 {
		t.Errorf("got (%v, %v), want (4.5, 12)", avg, count)
	}
	if gotPath != "/analytics/ratings/drivers/driver-1" {
		t.Errorf("got path %q, want /analytics/ratings/drivers/driver-1", gotPath)
	}
}

func TestAnalyticGateway_GetDriverRatingStats_NoRatingsYet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"average":0,"count":0}`))
	}))
	defer srv.Close()

	gw := NewAnalyticGateway(srv.URL, nil)
	avg, count, err := gw.GetDriverRatingStats(context.Background(), "driver-1")
	if err != nil {
		t.Fatalf("GetDriverRatingStats() error = %v, want nil", err)
	}
	if avg != 0 || count != 0 {
		t.Errorf("got (%v, %v), want (0, 0)", avg, count)
	}
}

func TestAnalyticGateway_GetDriverRatingStats_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	gw := NewAnalyticGateway(srv.URL, nil)
	_, _, err := gw.GetDriverRatingStats(context.Background(), "driver-1")
	if err == nil {
		t.Fatal("expected an error for an unexpected status code")
	}
}

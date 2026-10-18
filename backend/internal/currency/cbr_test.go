package currency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestCBRClientConvertsUsingNominalAndRequestedDate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("date_req"); got != "18/10/2026" {
			t.Errorf("date_req = %q, want 18/10/2026", got)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<ValCurs Date="17.10.2026">
  <Valute>
    <CharCode>USD</CharCode>
    <Nominal>1</Nominal>
    <Value>81,2500</Value>
  </Valute>
  <Valute>
    <CharCode>JPY</CharCode>
    <Nominal>100</Nominal>
    <Value>54,5000</Value>
  </Valute>
</ValCurs>`))
	}))
	defer server.Close()

	client := newCBRClient(server.URL, server.Client())
	date := time.Date(2026, time.October, 18, 0, 0, 0, 0, time.UTC)
	conversion, err := client.ToRubles(
		context.Background(),
		decimal.NewFromInt(1_000),
		"jpy",
		date,
	)
	if err != nil {
		t.Fatalf("ToRubles() returned an error: %v", err)
	}
	if !conversion.AmountRUB.Equal(decimal.RequireFromString("545")) {
		t.Errorf("AmountRUB = %s, want 545", conversion.AmountRUB)
	}
	if !conversion.Rate.Equal(decimal.RequireFromString("0.545")) {
		t.Errorf("Rate = %s, want 0.545", conversion.Rate)
	}
	if got := conversion.RateDate.Format("2006-01-02"); got != "2026-10-17" {
		t.Errorf("RateDate = %s, want 2026-10-17", got)
	}
}

func TestCBRClientReturnsNotFoundForUnknownCurrency(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><ValCurs Date="18.10.2026"></ValCurs>`))
	}))
	defer server.Close()

	client := newCBRClient(server.URL, server.Client())
	_, err := client.ToRubles(
		context.Background(),
		decimal.NewFromInt(10),
		"XYZ",
		time.Date(2026, time.October, 18, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("ToRubles() returned nil error for an unknown currency")
	}
}

func TestCBRClientDoesNotRequestRatesForRubles(t *testing.T) {
	t.Parallel()

	client := newCBRClient("::invalid URL::", http.DefaultClient)
	conversion, err := client.ToRubles(
		context.Background(),
		decimal.RequireFromString("123.456"),
		"RUB",
		time.Date(2026, time.October, 18, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("ToRubles() returned an error: %v", err)
	}
	if !conversion.AmountRUB.Equal(decimal.RequireFromString("123.46")) {
		t.Errorf("AmountRUB = %s, want 123.46", conversion.AmountRUB)
	}
}

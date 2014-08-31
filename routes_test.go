package main

import (
	"testing"

	"github.com/nictuku/eliteprofit/emdn"
)

type routeTest struct {
	station string
	route   Route
}

func TestBestBuy(t *testing.T) {
	count := 0
	store := newMarketStore()
	c := emdn.TestSubscribe()
	for m := range c {
		store.record(m.Transaction)
		count += 1
	}
	if count == 0 {
		t.Fatalf("Didn't receive any transactions from the test input.")
	}
	tests := []routeTest{
		{"Eranin (AZEBAN CITY)", Route{Profit: 287, DestinationStation: "Ross 1057 (Wang Estate)"}},
		{"Asellus Primus (BEAGLE 2 LANDING)", Route{Profit: 781, DestinationStation: "Nang Ta-khian (Hay Point)"}},
		{"LHS 3262 (Louis de Lacaille Prospect)", Route{Profit: 1018, DestinationStation: "Nang Ta-khian (Hay Point)"}},
		{"Bogus", Route{Profit: 0}},
	}

	for _, testStation := range tests {
		// Find the most profitable routes.
		routes := store.bestBuy(testStation.station, 2000000, 100)
		if len(routes) == 0 {
			t.Fatalf("nope: got %d wanted > 0 ", len(routes))
		}
		// Only the first is filled for now.
		r := routes[0]
		if r.DestinationStation != testStation.route.DestinationStation {
			t.Errorf("testStation %v: destination %v (wanted %v)\n",
				testStation, r.DestinationStation, testStation.route.DestinationStation)
		}
		if r.Profit != testStation.route.Profit {
			t.Errorf("testStation %v: price %+v (wanted %v)\n", testStation, r, testStation.route.Profit)
		}
	}
	t.Logf("woot")
}

func TestDistance(t *testing.T) {
	d := distance("Asellus Primus", "Eranin")
	want := 4.482060596143252
	if d != want {
		t.Errorf("distance between Asellus Primus and Eranin, got %v, wanted %v", d, want)
	}
}

func TestStarName(t *testing.T) {
	station := "i Bootis (CHANGO DOCK)"
	want := "i Bootis"

	starName := star(station)
	if starName != want {
		t.Errorf("star name got %q wanted %q", starName, want)
	}
}

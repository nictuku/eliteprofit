package main

import (
	"reflect"
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
	c, err := emdn.TestSubscribe()
	if err != nil {
		t.Fatal(err)
	}
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

func TestRoute(t *testing.T) {
	var tests = []struct {
		from      string
		to        string
		jumpRange float64
		want      []string
	}{
		// Close neighbors.
		{"Asellus Primus", "Eranin", 9999, []string{"Asellus Primus", "Eranin"}},

		// 29LY distance.
		{"Dahan", "Ovid", 9999, []string{"Dahan", "Ovid"}},
		{"Dahan", "Ovid", 6.1, []string{"Dahan", "Asellus Primus", "Eranin", "i Bootis", "Styx", "Opala", "Ovid"}},

		{"Asellus Primus", "Nang Ta-khian", 6.1, []string{"Asellus Primus", "LHS 3006", "G 239-25", "Nang Ta-khian"}},
	}
	for _, r := range tests {
		route := starRoute(r.from, r.to, r.jumpRange)
		if !reflect.DeepEqual(route, r.want) {
			t.Errorf("starRoute(%q, %q) = %q; want %q", r.from, r.to, route, r.want)
		}
	}
}

func BenchmarkStarRoute(b *testing.B) {
	// Build routing plan from Eranin to all known stars (as of Beta 1).
	// For the jump range, consider a Viper with full load (cargo and
	// fuel), or 9.16 LY according to http://changodock.com.
	star := "Eranin"
	for i := 0; i < b.N; i++ {
		for dest := range locs {
			if dest == star {
				continue
			}
			starRoute(star, dest, 9.16)
		}
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

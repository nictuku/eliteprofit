package main

import (
	"log"
	"math"
	"strings"

	"code.google.com/p/gos2/r3"
)

type Route struct {
	Item               string
	SourceStation      string
	BuyPrice           float64
	DestinationStation string
	SellPrice          float64
	Profit             float64
	Distance           float64
	JumpRange          float64
	Jumps              float64
}

// localItems finds all items with positive supply from a station that cost up
// to creditLimit.
func (s marketStore) localItems(station string, creditLimit float64) (items []Route) {
	for item, price := range s.stationSupply[station] {
		if price <= creditLimit {
			items = append(items, Route{Item: item, BuyPrice: price})
		}
	}
	return items
}

// bestBuy finds the route with maximum profit based on arguments. It currently
// only considers buying from local items and assumes a uniform travel cost -
// i.e: assumes that all systems are one jump away.
func (s marketStore) bestBuy(currentStation string, creditLimit float64, jumpRange float64) (routes []Route) {
	// Find top profit for each item.
	var bestProfit, profit float64

	var bestRoute Route
	for _, item := range s.localItems(currentStation, creditLimit) {
		// TODO: Consider distance and cargoLimit.
		bestPrice := s.maxDemand(item.Item)
		profit = bestPrice.SellPrice - item.BuyPrice
		if profit > bestProfit {
			d := distance(currentStation, bestPrice.Station)
			bestRoute = Route{
				Item:               item.Item,
				SourceStation:      currentStation,
				BuyPrice:           item.BuyPrice,
				DestinationStation: bestPrice.Station,
				SellPrice:          bestPrice.SellPrice,
				Profit:             profit,
				Distance:           d,
				JumpRange:          jumpRange,
				Jumps:              math.Ceil(d / jumpRange),
			}
			bestProfit = profit
		}
		// TODO: Consider the cargo limit.
	}
	// TODO: More routes.
	routes = []Route{bestRoute}
	log.Printf("Candidate best profit: deliver %v to %v for %v\n",
		bestRoute.Item, bestRoute.DestinationStation, bestRoute.Profit)
	return routes
}

// Names from 'i Bootis (CHANGO DOCK)' to 'i Bootis'
func star(station string) string {
	return strings.Split(station, " (")[0]
}

func distance(stationA, stationB string) float64 {
	// Input can be in the form "i Bootis (CHANGO DOCK)". Need to obtain the star name.
	return locs[star(stationA)].Distance(locs[star(stationB)])
}

func starRoute(from, to string, jumpRange float64) []string {
	search := make(map[string]bool, len(locs))
	for star := range locs {
		search[star] = true
	}
	return route(from, to, jumpRange, search)
}
func route(from, to string, jumpRange float64, search map[string]bool) []string {
	fromLoc, ok := locs[from]
	if !ok {
		return nil
	}
	toLoc, ok := locs[to]
	if !ok {
		return nil
	}
	// Are they reachable in one jump?
	fromDistance := fromLoc.Distance(toLoc)
	if fromDistance <= jumpRange {
		return []string{from, to}
	}

	// Use a brute-force method for now. Find the star closest to
	// the destination than the starting point.
	closest := from
	var distance float64 = 0
	// TODO: Reduce locs on each run.
	for star := range search {
		if star == to {
			continue
		}
		loc := locs[star]
		d := loc.Distance(toLoc)
		if fromLoc.Distance(loc) > fromDistance {
			// log.Printf("deleted %v", star)
			// delete(search, star)
			continue
		}
		if d < jumpRange {
			// Prefer the longest jump within range.
			if d > distance {
				closest = star
				distance = d
			}
		}
	}
	// log.Printf("from %v to %v diving into %v (range %v, distance %v)", from, to, closest, jumpRange, distance)
	return append(route(from, closest, jumpRange, search), to)
}

// Distances from http://forums.frontier.co.uk/showthread.php?t=34824
// Converted using https://gist.github.com/nictuku/46919118addfa5912f47.
var locs = map[string]r3.Vector{
	"Acihaut":        r3.Vector{-18.500000, 25.281250, -4.000000},
	"Aganippe":       r3.Vector{-11.562500, 43.812500, 11.625000},
	"Asellus Primus": r3.Vector{-23.937500, 40.875000, -1.343750},
	"Aulin":          r3.Vector{-19.687500, 32.687500, 4.750000},
	"Aulis":          r3.Vector{-16.468750, 44.187500, -11.437500},
	"BD+47 2112":     r3.Vector{-14.781250, 33.468750, -0.406250},
	"BD+55 1519":     r3.Vector{-16.937500, 44.718750, -16.593750},
	"Bolg":           r3.Vector{-7.906250, 34.718750, 2.125000},
	"Chi Herculis":   r3.Vector{-30.750000, 39.718750, 12.781250},
	"CM Draco":       r3.Vector{-35.687500, 30.937500, 2.156250},
	"Dahan":          r3.Vector{-19.750000, 41.781250, -3.187500},
	"DN Draconis":    r3.Vector{-27.093750, 21.625000, 0.781250},
	"DP Draconis":    r3.Vector{-17.500000, 25.968750, -11.375000},
	"Eranin":         r3.Vector{-22.843750, 36.531250, -1.187500},
	"G 239-25":       r3.Vector{-22.687500, 25.812500, -6.687500},
	"GD 319":         r3.Vector{-19.375000, 43.625000, -12.750000},
	"h Draconis":     r3.Vector{-39.843750, 29.562500, -3.906250},
	"Hermitage":      r3.Vector{-28.750000, 25.000000, 10.437500},
	"i Bootis":       r3.Vector{-22.375000, 34.843750, 4.000000},
	"Ithaca":         r3.Vector{-8.093750, 44.937500, -9.281250},
	"Keries":         r3.Vector{-18.906250, 27.218750, 12.593750},
	"Lalande 29917":  r3.Vector{-26.531250, 22.156250, -4.562500},
	"LFT 1361":       r3.Vector{-38.781250, 24.718750, -0.500000},
	"LFT 880":        r3.Vector{-22.812500, 31.406250, -18.343750},
	"LFT 992":        r3.Vector{-7.562500, 42.593750, 0.687500},
	"LHS 2819":       r3.Vector{-30.500000, 38.562500, -13.437500},
	"LHS 2884":       r3.Vector{-22.000000, 48.406250, 1.781250},
	"LHS 2887":       r3.Vector{-7.343750, 26.781250, 5.718750},
	"LHS 3006":       r3.Vector{-21.968750, 29.093750, -1.718750},
	"LHS 3262":       r3.Vector{-24.125000, 18.843750, 4.906250},
	"LHS 417":        r3.Vector{-18.312500, 18.187500, 4.906250},
	"LHS 5287":       r3.Vector{-36.406250, 48.187500, -0.781250},
	"LHS 6309":       r3.Vector{-33.562500, 33.125000, 13.468750},
	"LP 271-25":      r3.Vector{-10.468750, 31.843750, 7.312500},
	"LP 275-68":      r3.Vector{-23.343750, 25.062500, 15.187500},
	"LP 64-194":      r3.Vector{-21.656250, 32.218750, -16.218750},
	"LP 98-132":      r3.Vector{-26.781250, 37.031250, -4.593750},
	"Magec":          r3.Vector{-32.875000, 36.156250, 15.500000},
	"Meliae":         r3.Vector{-17.312500, 49.531250, -1.687500},
	"Morgor":         r3.Vector{-15.250000, 39.531250, -2.250000},
	"Nang Ta-khian":  r3.Vector{-18.218750, 26.562500, -6.343750},
	"Naraka":         r3.Vector{-34.093750, 26.218750, -5.531250},
	"Opala":          r3.Vector{-25.500000, 35.250000, 9.281250},
	"Ovid":           r3.Vector{-28.062500, 35.156250, 14.812500},
	"Pi-fang":        r3.Vector{-34.656250, 22.843750, -4.593750},
	"Rakapila":       r3.Vector{-14.906250, 33.625000, 9.125000},
	"Ross 1015":      r3.Vector{-6.093750, 29.468750, 3.031250},
	"Ross 1051":      r3.Vector{-37.218750, 44.500000, -5.062500},
	"Ross 1057":      r3.Vector{-32.312500, 26.187500, -12.437500},
	"Styx":           r3.Vector{-24.312500, 37.750000, 6.031250},
	"Surya":          r3.Vector{-38.468750, 39.250000, 5.406250},
	"Tilian":         r3.Vector{-21.531250, 22.312500, 10.125000},
	"WISE 1647+5632": r3.Vector{-21.593750, 17.718750, 1.750000},
	"Wyrd":           r3.Vector{-11.625000, 31.531250, -3.937500},
}

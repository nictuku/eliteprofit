package main

import "log"

type Route struct {
	Item               string
	SourceStation      string
	BuyPrice           float64
	DestinationStation string
	SellPrice          float64
	Profit             float64
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
func (s marketStore) bestBuy(currentStation string, creditLimit float64, cargoLimit int) (routes []Route) {
	// Find top profit for each item.
	var bestProfit, profit float64

	var bestRoute Route
	for _, item := range s.localItems(currentStation, creditLimit) {
		// TODO: Consider distance and cargoLimit.
		bestPrice := s.maxDemand(item.Item)
		profit = bestPrice.SellPrice - item.BuyPrice
		if profit > bestProfit {
			bestRoute = Route{
				Item:               item.Item,
				SourceStation:      currentStation,
				BuyPrice:           item.BuyPrice,
				DestinationStation: bestPrice.Station,
				SellPrice:          bestPrice.SellPrice,
				Profit:             profit,
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

package main

import (
	"fmt"

	"github.com/petar/GoLLRB/llrb"
)

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
	// TODO: Make this faster. Keep a map of station name to item prices.
	//
	// Currently it ranges over the entire marketStore map and filters
	// non-Supply items. For each item type, it traverses the supply search
	// tree until it finds the entry related to the current station.
	// Finally, record the prices and later returns them.
	for k, tree := range s {
		if k.Type != "Supply" {
			continue
		}
		pivot := suptrans{BuyPrice: creditLimit, Supply: 1}
		tree.DescendLessOrEqual(pivot, func(i llrb.Item) bool {
			item := i.(suptrans)
			if item.StationName != station {
				return true
			}
			items = append(items, Route{Item: item.ItemName, BuyPrice: item.BuyPrice})
			// Found our station.
			return false
		})
	}
	return items
}

// bestBuy finds the route with maximum profit based on arguments. It currently
// assumes a uniform travel cost - i.e: consider all systems are one jump away.
func (s marketStore) bestBuy(currentStation string, creditLimit float64, cargoLimit int) (routes []Route) {
	// - TODO: faster look-up of local items, ranked by lowest price.

	// Find top profit for each item.
	var bestProfit, profit float64
	var best demtrans
	for _, item := range s.localItems(currentStation, creditLimit) {
		// TODO: Consider distance and cargoLimit.
		bestPrice := s.maxDemand(item.Item)
		profit = bestPrice.SellPrice - item.BuyPrice
		if profit > bestProfit {
			bestProfit = profit
			best = bestPrice
		}
	}
	routes = []Route{{
		Item:               best.ItemName,
		SourceStation:      currentStation,
		DestinationStation: best.StationName,
		Profit:             bestProfit}}
	fmt.Printf("Candidate best profit: deliver %v to %v for %v\n", best.ItemName, best.StationName, bestProfit)
	return routes
}

// eliteprofit shows strategies that maximize trading profits in Elite: Dangerous,
// based on real-time market data.
package main

import (
	"flag"
	"fmt"
	"math"
	"net/http"

	"github.com/nictuku/eliteprofit/emdn"
	"github.com/petar/GoLLRB/llrb"
)

// References:
// - EMDN http://forums.frontier.co.uk/showthread.php?t=23585
// - distances: http://forums.frontier.co.uk/showthread.php?t=34824
var testMode = flag.Bool("testMode", false, "test mode, uses the input from data/input.json")

// Planned features:
//
// - bestBuyingPrice(currentLocation, creditLimit, shipType)
//   ~ answers the question "I'm in I Boots, what should I buy?"
//
/// - bestSellingPrice(currentLocation, product, shipType):
//   ~ "I'm in I Boots with a cargo of Gold in a Sidewinder, where should I sell it?"
//
// - bestRouteFrom(location string, creditLimit int) (item string, destination string) {
//

type Key struct {
	Type, Item string
}

type marketStore map[Key]*llrb.LLRB

const maxItems = 100

func (s marketStore) record(m emdn.Transaction) {
	k := Key{"Demand", m.ItemName}
	tree, ok := s[k]
	if !ok {
		tree = llrb.New()
		s[k] = tree
	}
	tree.InsertNoReplace(demtrans(m))
	for tree.Len() > maxItems {
		tree.DeleteMin()
	}

	k = Key{"Supply", m.ItemName}
	tree, ok = s[k]
	if !ok {
		tree = llrb.New()
		s[k] = tree
	}
	if m.BuyPrice == 0 {
		m.BuyPrice = math.MaxInt64
	}
	tree.InsertNoReplace(suptrans(m))
	for tree.Len() > maxItems {
		tree.DeleteMax()
	}
}

func (s marketStore) maxDemand(item string) demtrans {
	i := s[Key{"Demand", item}].Max()
	if i != nil {
		return i.(demtrans)
	}
	return demtrans{}
}

func (s marketStore) minSupply(item string) suptrans {
	i := s[Key{"Supply", item}].Min()
	if i != nil {
		return i.(suptrans)
	}
	return suptrans{}
}

func (s marketStore) buyHandler(w http.ResponseWriter, r *http.Request) {
	for k, _ := range s {
		if k.Type == "Demand" {
			continue
		}
		bestPrice := s.minSupply(k.Item)
		p := fmt.Sprintf("%v CR", bestPrice.BuyPrice)
		if bestPrice.BuyPrice == math.MaxInt64 {
			p = "N/A"
		}
		fmt.Fprintf(w, "%v: best place to buy from: %v, for %v (supply %v)\n", k.Item, bestPrice.StationName, p, bestPrice.Supply)
	}
}

func (s marketStore) sellHandler(w http.ResponseWriter, r *http.Request) {
	for k, _ := range s {
		if k.Type == "Supply" {
			continue
		}
		bestPrice := s.maxDemand(k.Item)
		p := fmt.Sprintf("%v CR", bestPrice.SellPrice)
		if bestPrice.BuyPrice == math.MaxInt64 {
			p = "N/A"
		}
		fmt.Fprintf(w, "%v: best place to sell to: %v, for %v (demand %v)\n", k.Item, bestPrice.StationName, p, bestPrice.Demand)
	}
}

type suptrans emdn.Transaction

func (t suptrans) Less(item llrb.Item) bool {
	if t.Supply == 0 {
		return false
	}
	return t.BuyPrice < item.(suptrans).BuyPrice
}

func (t suptrans) Type() string { return "Supply" }

type demtrans emdn.Transaction

func (t demtrans) Less(item llrb.Item) bool {
	if t.Demand == 0 {
		return false
	}
	return t.SellPrice < item.(demtrans).SellPrice
}

func (t demtrans) Type() string { return "Demand" }

func main() {
	flag.Parse()
	var sub func() <-chan emdn.Message
	if *testMode {
		sub = emdn.TestSubscribe
	} else {
		sub = emdn.Subscribe
	}
	store := make(marketStore)

	http.HandleFunc("/buy", store.buyHandler)

	http.HandleFunc("/sell", store.sellHandler)

	go http.ListenAndServe(":8080", nil)
	c := sub()
	for {
		m := <-c
		store.record(m.Transaction)
		item := m.Transaction.ItemName
		fmt.Printf("top supply for %+v: %+v\n", item, store.minSupply(item))
		fmt.Printf("top demand for %+v: %+v\n", item, store.maxDemand(item))
		fmt.Printf("length: supply %v, demand %v\n", store[Key{"Supply", item}].Len(), store[Key{"Demand", item}].Len())
	}

}

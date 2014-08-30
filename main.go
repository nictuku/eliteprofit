// eliteprofit shows strategies that maximize trading profits in Elite: Dangerous,
// based on real-time market data.
package main

import (
	"flag"
	"fmt"
	"math"
	"net/http"
	"sort"

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
	Type string // Supply or Demand
	Item string // coffee, gold, etc.
}

type marketStore struct {
	itemSupply    map[string]*llrb.LLRB
	itemDemand    map[string]*llrb.LLRB
	stationSupply map[string]map[Key]suptrans
	stationDemand map[string]map[Key]demtrans
}

func newMarketStore() *marketStore {
	return &marketStore{itemSupply: make(map[string]*llrb.LLRB), itemDemand: make(map[string]*llrb.LLRB)}
}

const maxItems = 5

func (s marketStore) record(m emdn.Transaction) {
	k := m.ItemName
	tree, ok := s.itemDemand[k]
	if !ok {
		tree = llrb.New()
		s.itemDemand[k] = tree
	}
	tree.ReplaceOrInsert(demtrans(m))
	for tree.Len() > maxItems {
		tree.DeleteMin()
	}

	k = m.ItemName
	tree, ok = s.itemSupply[k]
	if !ok {
		tree = llrb.New()
		s.itemSupply[k] = tree
	}
	if m.BuyPrice == 0 {
		m.BuyPrice = math.MaxInt64
	}
	tree.ReplaceOrInsert(suptrans(m))
	for tree.Len() > maxItems {
		tree.DeleteMax()
	}
}

func (s marketStore) maxDemand(item string) demtrans {
	i := s.itemDemand[item].Max()
	if i != nil {
		return i.(demtrans)
	}
	return demtrans{}
}

func (s marketStore) minSupply(item string) suptrans {
	i := s.itemSupply[item].Min()
	if i != nil {
		return i.(suptrans)
	}
	return suptrans{}
}

func (s marketStore) sorted() []string {
	items := make([]string, 0, len(s.itemSupply)) // XXX
	for k, _ := range s.itemSupply {
		items = append(items, k)
	}
	sort.Strings(items)
	return items
}

func (s marketStore) bestBuyHandler(w http.ResponseWriter, r *http.Request) {
	for _, route := range s.bestBuy("Nang Ta-khian (Hay Point)", 10000, 10000) {
		fmt.Fprintf(w, "%v: best place to buy from: %v, for %v\n", route.Item, route.SourceStation, route.BuyPrice)
	}
}

func (s marketStore) buyHandler(w http.ResponseWriter, r *http.Request) {
	for _, item := range s.sorted() {
		bestPrice := s.minSupply(item)
		p := fmt.Sprintf("%v CR", bestPrice.BuyPrice)
		if bestPrice.BuyPrice == math.MaxInt64 {
			p = "N/A"
		}
		fmt.Fprintf(w, "%v: best place to buy from: %v, for %v (supply %v)\n", item, bestPrice.StationName, p, bestPrice.Supply)
	}
}

func (s marketStore) sellHandler(w http.ResponseWriter, r *http.Request) {
	for _, item := range s.sorted() {
		bestPrice := s.maxDemand(item)
		p := fmt.Sprintf("%v CR", bestPrice.SellPrice)
		if bestPrice.BuyPrice == math.MaxInt64 {
			p = "N/A"
		}
		fmt.Fprintf(w, "%v: best place to sell to: %v, for %v (demand %v)\n", item, bestPrice.StationName, p, bestPrice.Demand)
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
	// XXX: HTTP handlers and zeromq are racing.
	store := newMarketStore()

	http.HandleFunc("/bestbuy", store.bestBuyHandler)
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
	}

}

// eliteprofit shows strategies that maximize trading profits in Elite: Dangerous,
// based on real-time market data.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/nictuku/eliteprofit/emdn"
	"github.com/petar/GoLLRB/llrb"
)

var (
	test      = flag.Bool("test", false, "test mode, uses the input from data/input.json")
	readCache = flag.Bool("readCache", true, "read the local data cache from data/large.gz")
	port      = flag.String("port", ":8080", "HTTP port to listen to")
)

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

// References:
// - EMDN http://forums.frontier.co.uk/showthread.php?t=23585
// - distances: http://forums.frontier.co.uk/showthread.php?t=34824

// Logging policy:
// - STDOUT is reserved for optionally printing EMDN messages
// - all other messages should be printed with the log message, which ensures
// they are sent to STDERR.

type marketStore struct {
	sync.Mutex
	itemSupply map[string]*llrb.LLRB
	itemDemand map[string]*llrb.LLRB
	// station => item => price
	stationSupply map[string]map[string]float64
	stationDemand map[string]map[string]float64
}

func newMarketStore() *marketStore {
	return &marketStore{
		itemSupply:    make(map[string]*llrb.LLRB),
		itemDemand:    make(map[string]*llrb.LLRB),
		stationSupply: make(map[string]map[string]float64),
		stationDemand: make(map[string]map[string]float64),
	}
}

const maxItems = 5

func stationPriceUpdate(m map[string]map[string]float64, station string, item string, price float64) {
	stationPrices := m[station]
	if stationPrices == nil {
		m[station] = make(map[string]float64)
	}
	m[station][item] = price
}

func (s marketStore) record(m emdn.Transaction) {
	k := m.Item
	// Demand
	tree, ok := s.itemDemand[k]
	if !ok {
		tree = llrb.New()
		s.itemDemand[k] = tree
	}
	tree.ReplaceOrInsert(demtrans(m))
	for tree.Len() > maxItems {
		tree.DeleteMin()
	}
	stationPriceUpdate(s.stationDemand, m.Station, k, m.SellPrice)

	// Supply
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
	stationPriceUpdate(s.stationSupply, m.Station, k, m.BuyPrice)
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

func (s marketStore) bestBuyHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	stations := make([]string, 0, len(s.stationSupply))
	for station := range s.stationSupply {
		stations = append(stations, station)
	}
	sort.Strings(stations)
	crLimit, _ := strconv.ParseFloat(r.FormValue("cr"), 64)
	if crLimit == 0 {
		crLimit = math.MaxFloat64
	}
	jumpRange, _ := strconv.ParseFloat(r.FormValue("jr"), 64)
	if jumpRange == 0 {
		jumpRange = math.MaxFloat64
	}

	for _, station := range stations {
		fmt.Fprintf(w, "======== buying from %v =======\n", station)
		for _, route := range s.bestBuy(station, crLimit, jumpRange) {
			fmt.Fprintf(w, "buy %v for %v and sell to %v for %v, profit %v\n", route.Item, route.BuyPrice, route.DestinationStation, route.SellPrice, route.Profit)
			fmt.Fprintf(w, "jumps %v, range %v, distance %v\n", route.Jumps, route.JumpRange, route.Distance)
		}
		fmt.Fprintf(w, "\n")
	}
}

func (s marketStore) buyHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	items := make([]string, 0, len(s.itemSupply))
	for station := range s.itemSupply {
		items = append(items, station)
	}
	sort.Strings(items)
	for _, item := range items {
		bestPrice := s.minSupply(item)
		p := fmt.Sprintf("%v CR", bestPrice.BuyPrice)
		if bestPrice.BuyPrice == math.MaxInt64 {
			p = "N/A"
		}
		fmt.Fprintf(w, "%v: best place to buy from: %v, for %v (supply %v)\n", item, bestPrice.Station, p, bestPrice.Supply)
	}
}

func (s marketStore) sellHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	items := make([]string, 0, len(s.itemDemand))
	for station := range s.itemDemand {
		items = append(items, station)
	}
	sort.Strings(items)
	for _, item := range items {
		bestPrice := s.maxDemand(item)
		p := fmt.Sprintf("%v CR", bestPrice.SellPrice)
		if bestPrice.BuyPrice == math.MaxInt64 {
			p = "N/A"
		}
		fmt.Fprintf(w, "%v: best place to sell to: %v, for %v (demand %v)\n", item, bestPrice.Station, p, bestPrice.Demand)
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

var mu sync.Mutex

func main() {
	flag.Parse()
	store := newMarketStore()

	var sub func() (<-chan emdn.Message, error)
	// XXX: HTTP handlers and zeromq are racing.
	if *test {
		sub = emdn.TestSubscribe
	} else {
		if *readCache {
			for m := range emdn.CacheRead() {
				store.record(m.Transaction)
			}
			log.Println("Cache read finished.")
		}
		sub = emdn.Subscribe
	}

	http.HandleFunc("/bestbuy", store.bestBuyHandler)
	http.HandleFunc("/buy", store.buyHandler)

	http.HandleFunc("/sell", store.sellHandler)
	go http.ListenAndServe(*port, nil)
	for {
		c, err := sub()
		if err != nil {
			log.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		for m := range c {
			mu.Lock()
			store.record(m.Transaction)
			mu.Unlock()
		}
		// c isn't expected to close unless in test mode. But if it
		// does, restart the subscription.
		time.Sleep(30 * time.Second)
	}

}

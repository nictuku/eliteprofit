package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/petar/GoLLRB/llrb"
)

// References:
// - EMDN http://forums.frontier.co.uk/showthread.php?t=23585
// - distances: http://forums.frontier.co.uk/showthread.php?t=34824

type Message struct {
	Transaction Transaction `json:"message"`
	Type        string      `json:"type"`
}

type Transaction struct {
	BuyPrice     float64 `json:"buyPrice"`
	CategoryName string  `json:"categoryName"`
	Demand       int     `json:"demand"`
	Supply       int     `json:"stationStock"`
	ItemName     string  `json:"itemName"`
	SellPrice    float64 `json:"sellPrice"`
	StationName  string  `json:"stationName"`
}

var testMode = flag.Bool("testMode", false, "test mode, uses the input from data/input.json")

// Queries:
// - bestSellingPrice(currentLocation, credits)
// - bestBuyingPrice(currentLocation, credits)

/* Goal:
func bestRouteFrom(location string, creditLimit int) (item string, destination string) {
	// find weighted shorted path between high supply and high demand.
}
*/

type Key struct {
	Type, Item string
}

type marketStore map[Key]*llrb.LLRB

const maxItems = 100

func (s marketStore) record(m Transaction) {
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

type suptrans Transaction

func (t suptrans) Less(item llrb.Item) bool {
	if t.Supply == 0 {
		return false
	}
	return t.BuyPrice < item.(suptrans).BuyPrice
}

func (t suptrans) Type() string { return "Supply" }

type demtrans Transaction

func (t demtrans) Less(item llrb.Item) bool {
	if t.Demand == 0 {
		return false
	}
	return t.SellPrice < item.(demtrans).SellPrice
}

func (t demtrans) Type() string { return "Demand" }

/*
{"message":{"buyPrice":0.0,"categoryName":"metals","demand":4509,"demandLevel":2
,"itemName":"uranium","sellPrice":2845.0,"stationName":"Asellus Primus (BEAGLE 2
 LANDING)","stationStock":0,"stationStockLevel":0,"timestamp":"2014-08-22T19:21:
38.503000+00:00"},"sender":"/QnobE//Oo86cZaJTT3c9YJ2N37hGi0YltWUArLxPUA=","signa
ture":"oqywduXExOwmBCzVIQD4rI0LbYTLZJyt8MmZGORku0HDO1qeX4/fkHiSibklO5KAuxWRan5YH
f553NgYwr//BQ==","type":"marketquote","version":"0.1"}
*/
func parseMessage(line string) (m Message) {
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		log.Fatal(err)
	}
	return m
}

func main() {
	flag.Parse()
	var scanner *bufio.Scanner
	if *testMode {
		f, err := os.Open(filepath.Join("data", "input.json"))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		scanner = bufio.NewScanner(f)
	} else {
		cmd := exec.Command(filepath.Join("c:", "marketdump", "firehose.exe"))
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}
		scanner = bufio.NewScanner(stdout)
	}
	store := make(marketStore)

	http.HandleFunc("/buy", store.buyHandler)

	http.HandleFunc("/sell", store.sellHandler)

	go http.ListenAndServe(":8080", nil)

	fmt.Println() // Println will add back the final '\n'

	for scanner.Scan() {
		line := scanner.Text()
		m := parseMessage(line)
		//fmt.Println(line)
		//	fmt.Printf("%+v\n", m)
		store.record(m.Transaction)
		item := m.Transaction.ItemName

		fmt.Printf("top supply for %+v: %+v\n", item, store.minSupply(item))
		fmt.Printf("top demand for %+v: %+v\n", item, store.maxDemand(item))
		fmt.Printf("length: supply %v, demand %v\n", store[Key{"Supply", item}].Len(), store[Key{"Demand", item}].Len())
	}
	select {}

}

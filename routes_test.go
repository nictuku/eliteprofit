package main

import "testing"

func TestBestBuy(t *testing.T) {
	store := make(marketStore)
	c := emdn.TestSubscribe()
	for m := <-c {
		store.record(m.Transaction)
		item := m.Transaction.ItemName
		fmt.Printf("top supply for %+v: %+v\n", item, store.minSupply(item))
		fmt.Printf("top demand for %+v: %+v\n", item, store.maxDemand(item))
	}
	t.Logf("woot")
}

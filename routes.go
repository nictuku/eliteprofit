package main

type Route struct {
	Item        string
	Source      string
	Destination string
	Profit      int
	// also distance, etc.
}

func (s marketStore) bestBuy(currentLocation string, creditLimit int, cargoLimit int) (items []string) {
	// Find the route with maximum profit based on arguments.
	//
	// Current limitations:
	// - assumes a uniform travel cost - i.e: consider all systems are one jump away.
	// -

	// Brute-force: for every item, find the local price and calculate the profit to all distances.
	// - TODO: faster look-up of local items, ranked by lowest price.
	// -
	var localItems map[string]int
	for k, trans := range s {
		if k.Type == "Demand" {
			// break
		}
		supply := trans.(suptrans)
		if supply.StationName == currentLocation {
			localItems[k] = supply.SellPrice
		}
	}

}

//  Elite Market Data Network subscriber
//
// See the *Eve* documentation about how this works.
// http://eve-market-data-relay.readthedocs.org/en/latest/using.html

package emdn

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	zmq "github.com/pebbe/zmq3"

	"fmt"
)

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

/*
{"message":{"buyPrice":0.0,"categoryName":"metals","demand":4509,"demandLevel":2
,"itemName":"uranium","sellPrice":2845.0,"stationName":"Asellus Primus (BEAGLE 2
 LANDING)","stationStock":0,"stationStockLevel":0,"timestamp":"2014-08-22T19:21:
38.503000+00:00"},"sender":"/QnobE//Oo86cZaJTT3c9YJ2N37hGi0YltWUArLxPUA=","signa
ture":"oqywduXExOwmBCzVIQD4rI0LbYTLZJyt8MmZGORku0HDO1qeX4/fkHiSibklO5KAuxWRan5YH
f553NgYwr//BQ==","type":"marketquote","version":"0.1"}
*/
func parseMessage(r io.Reader) (m Message) {
	dec := json.NewDecoder(r)
	if err := dec.Decode(&m); err != nil {
		log.Fatal(err)
	}
	return m
}

func TestSubscribe() <-chan Message {
	f, err := os.Open(filepath.Join("data", "input.json"))
	if err != nil {
		log.Fatal(err)
	}
	c := make(chan Message)
	go func() {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			buf := bytes.NewReader(scanner.Bytes())
			c <- parseMessage(buf)
			fmt.Println(".")
		}
		fmt.Println("finished processing input.json")
		if err := scanner.Err(); err != nil {
			fmt.Println(err)
		}
		select {}
	}()
	return c
}

func Subscribe() <-chan Message {
	receiver, _ := zmq.NewSocket(zmq.SUB)
	receiver.Connect("tcp://firehose.elite-market-data.net:9500")
	receiver.SetSubscribe("")
	c := make(chan Message)
	go func() {
		defer receiver.Close()
		for {
			// TODO: Find a way to avoid all the extra allocations.
			buf, err := receiver.RecvBytes(0)
			if err != nil {
				// XXX
				log.Fatal(err)
			}
			r, err := zlib.NewReader(bytes.NewReader(buf))
			if err != nil {
				// XXX
				log.Fatal(err)
			}
			c <- parseMessage(r)
			io.Copy(os.Stdout, r)
			r.Close()
		}
	}()
	return c
}

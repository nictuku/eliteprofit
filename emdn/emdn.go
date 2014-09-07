//  Elite Market Data Network subscriber
//
// See the *Eve* documentation about how this works.
// http://eve-market-data-relay.readthedocs.org/en/latest/using.html

package emdn

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
	zmq "github.com/pebbe/zmq2"

	"flag"
	"fmt"
)

var showLog = flag.Bool("showLog", false, "Show EMDN messages on the stdout, useful for backup")

type Message struct {
	Transaction Transaction `json:"message"`
	Type        string      `json:"type"`
}

type Transaction struct {
	BuyPrice  float64 `json:"buyPrice"`
	Category  string  `json:"categoryName"`
	Demand    int     `json:"demand"`
	Supply    int     `json:"stationStock"`
	Item      string  `json:"itemName"`
	SellPrice float64 `json:"sellPrice"`
	Station   string  `json:"stationName"`
}

/*
{"message":{"buyPrice":0.0,"categoryName":"metals","demand":4509,"demandLevel":2
,"itemName":"uranium","sellPrice":2845.0,"stationName":"Asellus Primus (BEAGLE 2
 LANDING)","stationStock":0,"stationStockLevel":0,"timestamp":"2014-08-22T19:21:
38.503000+00:00"},"sender":"/QnobE//Oo86cZaJTT3c9YJ2N37hGi0YltWUArLxPUA=","signa
ture":"oqywduXExOwmBCzVIQD4rI0LbYTLZJyt8MmZGORku0HDO1qeX4/fkHiSibklO5KAuxWRan5YH
f553NgYwr//BQ==","type":"marketquote","version":"0.1"}
*/

func CacheRead() <-chan Message {
	var f io.ReadCloser
	var err error
	if f, err = os.Open(filepath.Join("data", "large.gz")); err != nil {
		log.Fatal(err)
	}
	if f, err = gzip.NewReader(f); err != nil {
		log.Fatal(err)
	}
	return fileRead(f)
}

func TestSubscribe() (<-chan Message, error) {
	f, err := os.Open(filepath.Join("data", "input.json"))
	if err != nil {
		return nil, err
	}
	return fileRead(f), nil
}

func fileRead(f io.ReadCloser) <-chan Message {
	c := make(chan Message)
	go func() {
		defer f.Close()
		defer close(c)
		dec := json.NewDecoder(f)
		for {
			var m Message
			if err := dec.Decode(&m); err != nil {
				if err != io.EOF {
					log.Print("fileRead:", err)
				}
				break
			}
			c <- m
			fmt.Print("-")
		}
		fmt.Println("finished processing local file")
	}()
	return c
}

func Subscribe() (<-chan Message, error) {
	receiver, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf("zmq.NewSocket: %v", err)
	}
	if err = receiver.Connect("tcp://firehose.elite-market-data.net:9500"); err != nil {
		return nil, fmt.Errorf("receiver.Connect: %v", err)
	}
	if err := receiver.SetSubscribe(""); err != nil {
		return nil, fmt.Errorf("receiver.SetSubscribe: %v", err)
	}
	c := make(chan Message)

	go func() {
		for { // Start over if we have trouble.
			// TODO: Find a way to avoid all the extra allocations.
			msgs, err := receiver.RecvMessageBytes(0)
			if err != nil {
				log.Print(err)
				return
			}
			for _, buf := range msgs {
				r, err := zlib.NewReader(bytes.NewReader(buf))
				if err != nil {
					log.Print(err)
					return
				}
				var tee io.Reader = r
				if *showLog {
					tee = io.TeeReader(r, os.Stdout)
				}
				dec := json.NewDecoder(tee)
				for {
					var m Message
					if err = dec.Decode(&m); err != nil {
						if err != io.EOF {
							log.Print("Subscribe:", err)
							break
						}
					}
					c <- m
					fmt.Print(".")
				}
				r.Close()
			}
			log.Printf("Error: %v. Sleeping and trying again", err)
			time.Sleep(30 * time.Second)
			log.Println("Re-connecting")
		}
	}()
	return c, nil
}

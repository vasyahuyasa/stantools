package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/nats-io/stan.go"
)

var subjects = map[string]*uint64{}
var subjectOrder = []string{}

func main() {
	clientID := fmt.Sprintf("stanrate-%d", rand.Int())
	sc, err := stan.Connect(clusterID, clientID)
	if err != nil {
		log.Fatalf("can not connect to NATS server: %v", err)
	}
	defer sc.Close()

	// Unsubscribe
	sub, err := countRate("order", sc)
	if err != nil {
		log.Println("can not subscribe to %q: %v", "order", err)
		return
	}
	defer sub.Unsubscribe()

	go monitor(5)

	// TODO: wait for Ctrl+C
}

func countRate(subj string, sc stan.Conn) (stan.Subscription, error) {
	var counter *uint64
	*counter = 0
	subjects[subj] = counter
	subjectOrder = append(subjectOrder, subj)

	return sc.Subscribe(subj, func(m *stan.Msg) {
		atomic.AddUint64(counter, 1)
	})
}

func monitor(intervalSec int) {
	ticker := time.Tick(time.Second * time.Duration(intervalSec))

	for range ticker {
		for _, subj := range subjectOrder {
			counter := subjects[subj]
			c := atomic.SwapUint64(counter, 0)
			log.Printf("%s %d msg/%dsec", subj, c, intervalSec)
		}
	}
}

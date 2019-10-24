package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nats-io/stan.go"
)

const (
	defaultClusterID      = "nats"
	defaultClientIDPrefix = "stanrate"
)

var (
	subjects     = map[string]*uint64{}
	subjectOrder = []string{}
)

func main() {
	var (
		subjects, clusterID, clientID, natsUrl string
		interval                               int
		help                                   bool
	)

	flag.StringVar(&subjects, "subject", "", "Comma separated list of subjects")
	flag.StringVar(&clusterID, "cluster", defaultClusterID, "Cluster ID")
	flag.StringVar(&clientID, "client_id", fmt.Sprintf("%s-%d", defaultClientIDPrefix, rand.Intn(1000)), "Client ID")
	flag.StringVar(&natsUrl, "url", "nats://client:123456@localhost:4222", "NATS server url")
	flag.IntVar(&interval, "int", 5, "Interval in seconds between checks")
	flag.BoolVar(&help, "help", false, "This help text")

	flag.Parse()
	if help {
		flag.Usage()
	}

	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsUrl))
	if err != nil {
		log.Fatalf("can not connect to NATS server: %v", err)
	}
	defer sc.Close()

	// monitor each subject
	for _, s := range strings.Split(subjects, ",") {
		subject := strings.TrimSpace(s)

		sub, err := countRate(subject, sc)
		if err != nil {
			log.Printf("can not subscribe to %q: %v", subject, err)
			return
		}

		defer func() {
			log.Println("unsubscribe", subject)
			err := sub.Unsubscribe()
			if err != nil {
				log.Printf("can not unsubscribe %q: %v", subject, err)
			}
		}()
	}

	go monitor(interval)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)
	<-done

	log.Println("exit")

	// TODO: wait for Ctrl+C
}

func countRate(subj string, sc stan.Conn) (stan.Subscription, error) {
	var counter uint64
	subjects[subj] = &counter
	subjectOrder = append(subjectOrder, subj)

	return sc.Subscribe(subj, func(m *stan.Msg) {
		atomic.AddUint64(&counter, 1)
	})
}

func monitor(intervalSec int) {
	log.Printf("monitor %s every %d sec", strings.Join(subjectOrder, ", "), intervalSec)

	ticker := time.Tick(time.Second * time.Duration(intervalSec))

	for range ticker {
		for _, subj := range subjectOrder {
			counter := subjects[subj]
			c := atomic.SwapUint64(counter, 0)
			log.Printf("%s %d msg/%dsec", subj, c, intervalSec)
		}
	}
}

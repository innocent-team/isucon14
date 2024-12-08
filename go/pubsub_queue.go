package main

import (
	"context"
	"log"
	"time"
)

type MatchingPubSubQueue struct {
	queue chan *RideType
}

func NewMatchingPubSubQueue() *MatchingPubSubQueue {
	return &MatchingPubSubQueue{
		queue: make(chan *RideType, 100),
	}
}

func (q *MatchingPubSubQueue) Publish(ride *RideType) {
	log.Printf("queue: enqueueing msg=%#v, length=%d", ride, len(q.queue))
	q.queue <- ride
	log.Printf("queue: enqueud msg=%#v, length=%d", ride, len(q.queue))
}

func (q *MatchingPubSubQueue) Start(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	for {
		select {
		case <-tick.C:
			log.Printf("queue: length=%d", len(q.queue))
		case <-ctx.Done():
			return
		case ride := <-q.queue:
			log.Printf("queue: grabbed message=%#v, length=%d", ride, len(q.queue))
			for {
				// キューの先頭の人をマッチさせる
				// マッチするまでやりなおす
				missing, err := matcher(ctx, db, ride)
				if err != nil {
					log.Printf("failed to match: %v", err)
					return
				}
				if !missing {
					break
				}
			}
			log.Printf("queue: processed message=%#v, length=%d", ride, len(q.queue))
		}
	}
}

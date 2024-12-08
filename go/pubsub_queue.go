package main

import (
	"context"
	"log"
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
	for ride := range q.queue {
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

func (q *MatchingPubSubQueue) Close() {
	close(q.queue)
}

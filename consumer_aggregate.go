package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

type Metrics struct {
	Total     int64
	Status2xx int64
	Status4xx int64
	Status5xx int64

	Latencies []int
}

type Aggregator struct {
	sync.Mutex
	currentMinute time.Time
	metrics       Metrics
}

func NewAggregator() *Aggregator {
	return &Aggregator{
		currentMinute: truncateMinute(time.Now()),
	}
}

func truncateMinute(t time.Time) time.Time {
	return t.Truncate(time.Minute)
}

func (a *Aggregator) Add(status int, latency int) {
	a.Lock()
	defer a.Unlock()

	nowMin := truncateMinute(time.Now())

	if nowMin.After(a.currentMinute) {
		a.flush()
		a.currentMinute = nowMin
		a.metrics = Metrics{}
	}

	a.metrics.Total++
	a.metrics.Latencies = append(a.metrics.Latencies, latency)

	switch {
	case status >= 200 && status < 300:
		a.metrics.Status2xx++
	case status >= 400 && status < 500:
		a.metrics.Status4xx++
	case status >= 500:
		a.metrics.Status5xx++
	}
}

func percentile(values []int, p float64) int {
	if len(values) == 0 {
		return 0
	}

	// naive sort (ok para demo)
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	k := int(float64(len(values)-1) * p)
	return values[k]
}

func (a *Aggregator) flush() {
	m := a.metrics

	if m.Total == 0 {
		return
	}

	sum := 0
	for _, v := range m.Latencies {
		sum += v
	}

	avg := float64(sum) / float64(len(m.Latencies))
	p95 := percentile(m.Latencies, 0.95)

	fmt.Printf(
		"\n[%s]\nRequests=%d 2xx=%d 4xx=%d 5xx=%d avg=%.2fms p95=%dms\n\n",
		a.currentMinute.Format("15:04"),
		m.Total,
		m.Status2xx,
		m.Status4xx,
		m.Status5xx,
		avg,
		p95,
	)
}

type Handler struct {
	agg *Aggregator
}

func (Handler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (Handler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h Handler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		status, latency := parse(string(msg.Value))
		h.agg.Add(status, latency)
		sess.MarkMessage(msg, "")
	}
	return nil
}

func parse(line string) (int, int) {
	// request_id=1 path=/api status=200 latency=12
	parts := strings.Fields(line)

	status := 0
	latency := 0

	for _, p := range parts {
		if strings.HasPrefix(p, "status=") {
			status, _ = strconv.Atoi(strings.TrimPrefix(p, "status="))
		}
		if strings.HasPrefix(p, "latency=") {
			latency, _ = strconv.Atoi(strings.TrimPrefix(p, "latency="))
		}
	}

	return status, latency
}

func main() {
	brokers := []string{"localhost:9092"}
	group := "agg-consumer"
	topic := "http-logs"

	config := sarama.NewConfig()
	config.Version = sarama.V3_6_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer, err := sarama.NewConsumerGroup(brokers, group, config)
	if err != nil {
		log.Fatal(err)
	}

	agg := NewAggregator()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			err := consumer.Consume(ctx, []string{topic}, Handler{agg: agg})
			if err != nil {
				log.Println(err)
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	fmt.Println("closing")
}

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	brokers := []string{"localhost:9092"}
	topic := "http-logs"

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Return.Successes = false
	config.Producer.Flush.Frequency = 10 * time.Millisecond
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Flush.Messages = 5000
	config.Producer.MaxMessageBytes = 1_000_000

	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		log.Fatal(err)
	}

	defer producer.Close()

	// 500k/min ≈ 8333 msg/sec
	ratePerSecond := 8333
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	go func() {
		for err := range producer.Errors() {
			log.Println("producer error:", err)
		}
	}()

	counter := 0

	for range ticker.C {
		for i := 0; i < ratePerSecond; i++ {
			counter++
			msg := fmt.Sprintf("request_id=%d path=/api/test status=200 latency=12", counter)

			producer.Input() <- &sarama.ProducerMessage{
				Topic: topic,
				Value: sarama.StringEncoder(msg),
			}
		}

		fmt.Println("sent", ratePerSecond, "messages")
	}
}

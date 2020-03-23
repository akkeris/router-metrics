package main

import (
	"context"
	"fmt"
	"github.com/Shopify/sarama"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var conn net.Conn
var err error
var debugmode bool

func main() {
	envdebug := os.Getenv("DEBUG")
	if envdebug == "" {
		envdebug = "false"
	}
	debugmode, err = strconv.ParseBool(envdebug)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("debug mode: %v\n", debugmode)
	conn, err = net.Dial("tcp", os.Getenv("OPENTSDB_IP"))
	if err != nil {
		fmt.Println("dial error:", err)
		os.Exit(1)
	}

	version, err := sarama.ParseKafkaVersion("2.0.0")
	if err != nil {
		log.Panicf("Error parsing Kafka version: %v", err)
	}

	config := sarama.NewConfig()
	config.Version = version
	config.Consumer.Return.Errors = true
	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	fmt.Println(brokers)
	topics := []string{"alamoweblogs"}
	group := os.Getenv("CONSUMER_GROUP_NAME")
	consumer := Consumer{
		ready: make(chan bool),
	}
	ctx, cancel := context.WithCancel(context.Background())
	client, err := sarama.NewConsumerGroup(brokers, group, config)
	if err != nil {
		log.Panicf("Error creating consumer group client: %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, topics, &consumer); err != nil {
				log.Panicf("Error from consumer: %v", err)
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // Await till the consumer has been set up
	log.Println("Sarama consumer up and running!...")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		log.Println("terminating: context cancelled")
	case <-sigterm:
		log.Println("terminating: via signal")
	}
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		log.Panicf("Error closing client: %v", err)
	}
}

type Consumer struct {
	ready chan bool
}

func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {

	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for message := range claim.Messages() {
                if (debugmode){
		    log.Printf("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
                }
                sock(string(message.Value[:]))
		session.MarkMessage(message, "")
	}

	return nil
}
func sock(logline string) {
	words := strings.Fields(logline)
	var fieldmap map[string]string
	fieldmap = make(map[string]string)
	for _, element := range words {
		if strings.Contains(element, "=") {
			fieldmap[strings.Split(element, "=")[0]] = strings.Split(element, "=")[1]
		}
	}
	host := fieldmap["hostname"]
	if !(strings.HasPrefix(host, "alamotest")) && len(words) > 9 && !strings.Contains(strings.Join(words, " "), "4813") {
		status := fieldmap["status"]
		service := fieldmap["service"]
		connect := fieldmap["connect"]
		total := fieldmap["total"]
		site_domain := fieldmap["site_domain"]
		if debugmode {
			fmt.Println("pushing data for " + host)
		}
		metricname := "router.service.ms"
		timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
		value := service
		put := "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
		if debugmode {
			fmt.Println(put)
		}
		if !debugmode {
			fmt.Fprintf(conn, put)
		}

		metricname = "router.total.ms"
		value = total
		put = "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
		if debugmode {
			fmt.Println(put)
		}
		if !debugmode {
			fmt.Fprintf(conn, put)
		}
		metricname = "router.connect.ms"
		value = connect
		put = "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
		if debugmode {
			fmt.Println(put)
		}
		if !debugmode {
			fmt.Fprintf(conn, put)
		}

		metricname = "router.status." + status
		value = "1"
		put = "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
		if debugmode {
			fmt.Println(put)
		}
		if !debugmode {
			fmt.Fprintf(conn, put)
		}

		metricname = "router.requests.count"
		value = "1"
		put = "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
		if debugmode {
			fmt.Println(put)
		}
		if !debugmode {
			fmt.Fprintf(conn, put)
		}

		if site_domain != "" {
			host = site_domain
			if debugmode {
				fmt.Println("pushing data for " + host)
			}
			metricname := "router.service.ms"
			timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
			value := service
			put = "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
			if debugmode {
				fmt.Println(put)
			}
			if !debugmode {
				fmt.Fprintf(conn, put)
			}
			metricname = "router.total.ms"
			value = total
			put = "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
			if debugmode {
				fmt.Println(put)
			}
			if !debugmode {
				fmt.Fprintf(conn, put)
			}

			metricname = "router.connect.ms"
			value = connect
			put = "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
			if debugmode {
				fmt.Println(put)
			}
			if !debugmode {
				fmt.Fprintf(conn, put)
			}

			metricname = "router.status." + status
			value = "1"
			put = "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
			if debugmode {
				fmt.Println(put)
			}
			if !debugmode {
				fmt.Fprintf(conn, put)
			}

			metricname = "router.requests.count"
			value = "1"
			put = "put " + metricname + " " + timestamp + " " + value + " fqdn=" + host + "\n"
			if debugmode {
				fmt.Println(put)
			}
			if !debugmode {
				fmt.Fprintf(conn, put)
			}
		}

	}
	processed++

}

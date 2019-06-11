package main

import (
	"fmt"
	cluster "github.com/bsm/sarama-cluster"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

var conn net.Conn
var err error
var processed int
var debugmode bool

func main() {
	processed = 0
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

	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true
	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	topics := []string{"alamoweblogs"}
	consumer, err := cluster.NewConsumer(brokers, os.Getenv("CONSUMER_GROUP_NAME"), topics, config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer consumer.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	signal.Notify(signals, os.Kill)

	consumed := 0

	for {
		select {
		case msg, more := <-consumer.Messages():
			if more {
				if debugmode {
					fmt.Fprintf(os.Stdout, "%s/%d/%d\t%s\t%s\n", msg.Topic, msg.Partition, msg.Offset, msg.Key, msg.Value)
				}
				sock(string(msg.Value[:]))
				consumer.MarkOffset(msg, "") // mark message as processed
				consumed++
			}
		case err, more := <-consumer.Errors():
			if more {
				log.Printf("Error: %s\n", err.Error())
			}
		case ntf, more := <-consumer.Notifications():
			if more {
				log.Printf("Rebalanced: %+v\n", ntf)
			}
		case <-signals:
			return
		}
	}

	log.Printf("Consumed: %d\n", consumed)
	log.Printf("Processed: %d\n", processed)
	conn.Close()

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

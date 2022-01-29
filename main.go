package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	observe "github.com/rustomax/observe-common-go"
	"github.com/segmentio/kafka-go"
)

type Config struct {
	ApiUrl        string
	ExtraPath     string
	Customer      string
	Token         string
	Topic         string
	BrokerAddress string
	ConsumerGroup string
}

type Payload struct {
	Data json.RawMessage `json:"data"`
}

func main() {
	// Get config file name from command-line if provided,
	// otherwise use default
	config_file_path := "/etc/observe/default.json"
	if len(os.Args) >= 2 {
		config_file_path = os.Args[1]
	}

	// Read config file
	config, err := ReadConfig(config_file_path)
	if err != nil {
		log.Fatalf("ERROR: Failed to load config file: %v", err)
	} else {
		log.Printf("INFO: Loaded config file")
	}

	// Call the main consumer loop
	ctx := context.Background()
	consume(ctx, config)
}

func ReadConfig(config_path string) (Config, error) {
	config := Config{}
	config_file, err := os.Open(config_path)
	if err != nil {
		return config, err
	}
	defer config_file.Close()

	decoder := json.NewDecoder(config_file)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	fmt.Printf("%v\n", config)

	return config, nil
}

func consume(ctx context.Context, config Config) {
	l := log.New(os.Stdout, "INFO: Kafka reader: ", 0)

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{config.BrokerAddress},
		Topic:       config.Topic,
		StartOffset: kafka.LastOffset,
		GroupID:     config.ConsumerGroup,
		//MinBytes:    100,
		//MaxBytes:    1e6,
		//MaxWait:     10 * time.Second,
		Logger: l,
	})
	for {
		// ReadMessage blocks until we receive the next event
		msg, err := r.ReadMessage(ctx)

		if err != nil {
			log.Printf("ERROR: Failed to read message from Kafka: %v", err.Error())
			continue
		}
		// Convert Kafka message to JSON as expected by Observe HTTP collector API
		message := json.RawMessage(msg.Value)
		var payload Payload
		payload.Data, err = json.Marshal(&message)
		if err != nil {
			log.Printf("ERROR: Could not convert Kafka message to JSON: %v", err.Error())
			continue
		}

		// Send payload to Observe
		result, err := observe.SendPayload(payload, config.ApiUrl, config.ExtraPath, config.Customer, config.Token)
		if err != nil {
			log.Printf("ERROR: Failed to send json payload to Observe API: %v", err)
		} else {
			log.Printf("INFO: Sent data to Observe API; got response: %s", result)
		}
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/adjust/rmq"
	"github.com/streadway/amqp"
)

var config Config

type Config struct {
	Name    string `json:"name"`
	AppPath string `json:"app_path"`
	Queue   struct {
		Type     string `json:"type"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		DB       int    `json:"db"`
	} `json:"queue"`
	PidKill  bool `json:"pid_kill"`
	Commands struct {
		Start   string `json:"start"`
		Stop    string `json:"stop"`
		Restart string `json:"restart"`
		Reload  string `json:"reload"`
	} `json:"commands"`
}

type Consumer struct {
	name   string
}

func (consumer *Consumer) Consume(delivery rmq.Delivery) {
	
}

func LoadConfiguration(file string) Config {
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func ConsumeMsg(msg string, config Config) bool {
	switch m := strings.ToLower(msg); m {
	case "start":
		StartApp(config)
		break
	case "stop":
		StopApp(config)
		break
	case "restart":
		RestartApp(config)
		break
	case "reload":
		ReloadApp(config)
		break
	default:
		return false
	}
	return true
}

func HandleRabbitMQ(config Config) {
	dsn := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/",
		config.Queue.User,
		config.Queue.Password,
		config.Queue.Host,
		config.Queue.Port,
	)
	conn, err := amqp.Dial(dsn)
	FailOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	FailOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		config.Name, // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	FailOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	FailOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			if ConsumeMsg(fmt.Sprintf("%s", d.Body), config) {
				log.Printf("Received a message: %s", d.Body)
			} else {
				log.Printf("Unknown message: %s", d.Body)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func HandleRedis(config Config) {
	dsn := fmt.Sprintf("%s:%d", config.Queue.Host, config.Queue.Port)
	connection := rmq.OpenConnection("appman", "tcp", dsn, config.Queue.DB)
	taskQueue := connection.OpenQueue(config.Name)
	taskQueue.StartConsuming(10, time.Second)
	// taskQueue.AddConsumerFunc(func(delivery rmq.Delivery) {

	// })
}

func StartApp(config Config) {
	cmd := exec.Command(config.AppPath, config.Commands.Start)
	cmd.Stdout = os.Stdout
	err := cmd.Start()
	if err != nil {
		FailOnError(err, "Failed to start app")
	}
}

func StopApp(config Config) {
	if config.PidKill {

	} else {
		exec.Command(config.AppPath, config.Commands.Stop)
	}
}

func RestartApp(config Config) {
	if config.PidKill {
		StopApp(config)
		StartApp(config)
	} else {
		exec.Command(config.AppPath, config.Commands.Restart)
	}
}

func ReloadApp(config Config) {
	exec.Command(config.AppPath, config.Commands.Reload)
}

func main() {
	config := LoadConfiguration("./.appman.config.json")
	if strings.ToLower(config.Queue.Type) == "rabbitmq" {
		StartApp(config)
		HandleRabbitMQ(config)
	} else if strings.ToLower(config.Queue.Type) == "redis" {
		StartApp(config)
		HandleRedis(config)
	} else {
		fmt.Printf("Invalid queue type %s.\n", config.Queue.Type)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ramadani/starboost/internal/consumer"
	"github.com/ramadani/starboost/internal/transport"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Config struct {
	Address string `yaml:"address" validate:"required"`
	Debug   bool
	Kafka   struct {
		Addresses []string `yaml:"addresses" validate:"required"`
	}
	Consumer struct {
		Enabled bool     `yaml:"enabled"`
		GroupID string   `yaml:"groupId" validate:"required"`
		Topics  []string `yaml:"topics" validate:"required"`
		Output  struct {
			Stdout bool   `yaml:"stdout"`
			File   string `yaml:"file"`
		} `yaml:"output"`
	}
}

func main() {
	// config
	configFlagValue := flag.String("config", "config.yaml", "configuration file location")
	flag.Parse()

	conf, err := loadConfig(*configFlagValue)
	if err = validator.New().Struct(conf); err != nil {
		panic(fmt.Errorf("load config error: %q", err))
	}

	// logger
	logger, err := buildLogger(conf)
	if err != nil {
		panic(fmt.Errorf("fail init logger: %q", err))
	}

	// watermill
	var wmLogger watermill.LoggerAdapter
	if conf.Debug {
		wmLogger = watermill.NewStdLogger(conf.Debug, conf.Debug)
	}

	// publisher
	publisher, err := buildPublisher(conf, wmLogger)
	if err != nil {
		panic(fmt.Errorf("fail init publisher: %q", err))
	}

	// subscriber
	var subscriber *kafka.Subscriber

	if conf.Consumer.Enabled {
		subscriber, err = buildSubscriber(conf, wmLogger)
		if err != nil {
			panic(fmt.Errorf("fail init subscriber: %q", err))
		}

		consumerHandler := consumer.NewHandler(logger)

		for _, topic := range conf.Consumer.Topics {
			messages, err := subscriber.Subscribe(context.Background(), topic)
			if err != nil {
				panic(err)
			}

			go consumerHandler(topic, messages)
		}
	}

	// app handler
	appHandler := &transport.Handler{Publisher: publisher}

	// echo server
	e := echo.New()
	e.HidePort = true
	e.HideBanner = true
	e.Use(middleware.Recover())

	// app routes
	e.GET("/ping", appHandler.Ping)
	e.POST("/publish", appHandler.Publish)

	// Start server
	go func() {
		if err := e.Start(conf.Address); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	fmt.Printf("starboost started on %s\n", conf.Address)
	if conf.Consumer.Enabled {
		fmt.Println("consumer ready!")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	if conf.Consumer.Enabled {
		if err := subscriber.Close(); err != nil {
			log.Fatalln(err)
		}
	}
}

func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cant open file: %q", err)
	}

	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("cant read file: %q", err)
	}

	conf := new(Config)
	if err = yaml.Unmarshal(fileContent, &conf); err != nil {
		return nil, fmt.Errorf("cant unmarshal yaml file: %q", err)
	}
	if err = validator.New().Struct(conf); err != nil {
		return nil, fmt.Errorf("config validation error: %q", err)
	}
	return conf, nil
}

func buildLogger(conf *Config) (*zap.Logger, error) {
	outputPaths := make([]string, 0)
	out := conf.Consumer.Output

	if out.Stdout {
		outputPaths = append(outputPaths, "stdout")
	}
	if out.File != "" {
		outputPaths = append(outputPaths, out.File)
	}

	return zap.Config{
		Encoding:      "json",
		Level:         zap.NewAtomicLevelAt(zapcore.InfoLevel),
		OutputPaths:   outputPaths,
		DisableCaller: true,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,
		},
	}.Build()
}

func buildPublisher(conf *Config, logger watermill.LoggerAdapter) (*kafka.Publisher, error) {
	kafkaConf := kafka.PublisherConfig{
		Brokers:   conf.Kafka.Addresses,
		Marshaler: kafka.DefaultMarshaler{},
	}
	return kafka.NewPublisher(kafkaConf, logger)
}

func buildSubscriber(conf *Config, logger watermill.LoggerAdapter) (*kafka.Subscriber, error) {
	saramaSubscriberConfig := kafka.DefaultSaramaSubscriberConfig()
	saramaSubscriberConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	return kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               conf.Kafka.Addresses,
			Unmarshaler:           kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaSubscriberConfig,
			ConsumerGroup:         conf.Consumer.GroupID,
		},
		logger,
	)
}

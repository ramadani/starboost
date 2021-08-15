package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ramadani/starboost/internal/transport"
	"gopkg.in/yaml.v2"
	"io/ioutil"
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
}

func main() {
	configFlagValue := flag.String("config", "config.yaml", "configuration file location")

	flag.Parse()

	file, err := os.Open(*configFlagValue)
	if err != nil {
		panic(fmt.Errorf("cant open file: %q", err))
	}

	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		panic(fmt.Errorf("cant read file: %q", err))
	}

	conf := new(Config)

	err = yaml.Unmarshal(fileContent, &conf)
	if err != nil {
		panic(fmt.Errorf("cant unmarshal yaml file: %q", err))
	}

	if err = validator.New().Struct(conf); err != nil {
		panic(fmt.Errorf("config validation error: %q", err))
	}

	kafkaConf := kafka.PublisherConfig{
		Brokers:   conf.Kafka.Addresses,
		Marshaler: kafka.DefaultMarshaler{},
	}
	publisher, err := kafka.NewPublisher(kafkaConf, watermill.NewStdLogger(conf.Debug, conf.Debug))
	if err != nil {
		panic(fmt.Errorf("cant create publisher: %q", err))
	}

	appHandler := &transport.Handler{
		Publisher: publisher,
	}

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

	fmt.Printf("starboost started on \033[32m%s\033[32m\n", conf.Address)

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

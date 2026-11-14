package example

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/kastrul/gowsdl/example/gen"
	"github.com/kastrul/gowsdl/soap"
)

// ExampleBasicUsage calls a service through a generated client with default options.
func ExampleBasicUsage() {
	client := soap.NewClient("http://svc.asmx")
	service := gen.NewStockQuotePortType(client)
	reply, err := service.GetLastTradePrice(context.Background(), &gen.TradePriceRequest{})
	if err != nil {
		log.Fatalf("could't get trade prices: %v", err)
	}
	log.Println(reply)
}

// ExampleWithOptions calls a service through a generated client with a timeout,
// basic auth and a custom TLS configuration.
func ExampleWithOptions() {
	client := soap.NewClient(
		"http://svc.asmx",
		soap.WithTimeout(time.Second*5),
		soap.WithBasicAuth("usr", "psw"),
		soap.WithTLS(&tls.Config{MinVersion: tls.VersionTLS12}),
	)
	service := gen.NewStockQuotePortType(client)
	reply, err := service.GetLastTradePrice(context.Background(), &gen.TradePriceRequest{})
	if err != nil {
		log.Fatalf("could't get trade prices: %v", err)
	}
	log.Println(reply)
}

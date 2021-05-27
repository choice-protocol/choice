package main

import (
	"bytes"
	"context"
	"encoding/json"

	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/mitchellh/hashstructure/v2"
	// "cloud.google.com/go/bigquery"
)

// Item represents a row item. the auction is initially just open or closed, but later on different kinds of openings (time bundled, solo, etc)
type LogEntry struct {
	paramsHash string
	Payload    map[string]interface{}
	Auction    string
	timestamp  time.Time
}

// Parse the requests body
func parseRequestBody(request *http.Request) map[string]interface{} {

	// Read body to buffer
	body, err := ioutil.ReadAll(request.Body)

	if err != nil {
		log.Printf("Error reading body: %v", err)
		panic(err)
	}

	var requestPayload map[string]interface{}
	err = json.Unmarshal([]byte(body), &requestPayload)

	if err != nil {
		log.Printf("Error reading body: %v", err)
		panic(err)
	}

	// Because go lang is a pain in the ass if you read the body then any susequent calls
	// are unable to read the body again....
	request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return requestPayload
}

func debugHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "debugging")
}

type Saver = func(LogEntry) error

// Generate a function which embeds a firestore client and allows saving logEntries to firestore
func genSaver(client *firestore.Client, ctx context.Context) Saver {
	return func(logEntry LogEntry) error {
		collection := client.Collection("txs").Doc(logEntry.paramsHash)
		_, err := collection.Create(ctx, logEntry)
		if err != nil {
			return fmt.Errorf("Failed to add transaction: %v", err)
		}

		return nil
	}
}

type RequestHandler = func(http.ResponseWriter, *http.Request)

func genRequestHandler(vanillaURL *url.URL, bidderURL *url.URL, saver Saver) RequestHandler {
	log.Printf("Starting bidder proxy: %s", bidderURL)
	bidderProxy := httputil.NewSingleHostReverseProxy(bidderURL)

	log.Printf("Starting vanilla proxy: %s", vanillaURL)
	vanillaProxy := httputil.NewSingleHostReverseProxy(vanillaURL)

	return func(res http.ResponseWriter, req *http.Request) {
		requestPayload := parseRequestBody(req)
		log.Printf("Request: %+v\n", requestPayload)
		if requestPayload["method"] == "eth_sendRawTransaction" ||
			requestPayload["method"] == "eth_sendTransaction" ||
			requestPayload["method"] == "eth_sendRawTransaction_reserve" ||
			requestPayload["method"] == "eth_sendTransaction_reserve" {
			// this we want to keep, build and save log
			objectHash, err := hashstructure.Hash(requestPayload["params"], hashstructure.FormatV2, nil)
			objectHashString := strconv.FormatUint(objectHash, 10)

			if err != nil {
				log.Panicf("%d", err)
			}
			// default headers
			req.Header.Set("X-Choice-Operator-Version", "0.01")

			logItem := LogEntry{paramsHash: objectHashString, Payload: requestPayload, timestamp: time.Now(), Auction: "open"}

			err = saver(logItem)
			if err != nil {
				log.Printf("Failed to record transaction: %s", err)
			} else {
				log.Printf("Successfully recorded transaction")
			}
			req.Host = bidderURL.Host
			bidderProxy.ServeHTTP(res, req)
		} else {
			// Update the headers to allow for SSL redirection
			req.URL.Host = vanillaURL.Host
			req.URL.Scheme = vanillaURL.Scheme
			// req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
			req.Header.Set("X-Choice-Operator-Version", "0.01")
			req.Host = vanillaURL.Host

			// Note that ServeHttp is non blocking and uses a go routine under the hood
			vanillaProxy.ServeHTTP(res, req)
		}
	}
}

func main() {
	log.Print("starting server...")

	projectId := os.Getenv("CHOICE_PROJECT_ID")
	// Setup client
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectId)
	if err != nil {
		log.Fatalf("Fatal firebase error :%s", err)
	}
	defer client.Close()

	saver := genSaver(client, ctx)

	// bidderProxy
	bidderURL, err := url.Parse(os.Getenv("CHOICE_BIDDER_URL"))
	if err != nil {
		log.Fatalf("Set environment variable CHOICE_BIDDER_URL")
	}

	// vanillaProxy
	vanillaURL, err := url.Parse(os.Getenv("CHOICE_VANILLA_URL"))
	if err != nil {
		log.Fatalf("Set environment variable CHOICE_VANILLA_URL")
	}

	handler := genRequestHandler(vanillaURL, bidderURL, saver)

	// start server
	http.HandleFunc("/", handler)
	// start server
	http.HandleFunc("/debug", debugHandler)

	// Determine port for HTTP service.
	port := os.Getenv("CHOICE_PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

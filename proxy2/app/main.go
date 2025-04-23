package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"http-proxy/pkg/api"
	"http-proxy/pkg/proxy"
	"http-proxy/repo"
	"http-proxy/server"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func mongoConnect(username, password, host string, port int) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connectionString := fmt.Sprintf("mongodb://%s:%s@%s:%d", username, password, host, port)
	return mongo.Connect(ctx, options.Client().ApplyURI(connectionString))
}

func main() {
	mongoConn, err := mongoConnect("root", "example", "mongo", 27017)
	if err != nil {
		log.Fatal(err)
	}

	requests := repo.NewMongoRequestSaver(mongoConn)
	responses := repo.NewMongoResponseSaver(mongoConn)

	handler, err := proxy.NewHandler(requests, responses)
	if err != nil {
		log.Fatal(err)
	}

	go startHttpApi(requests, responses)

	server.Run(8080, handler.Handle)
}

func startHttpApi(req repo.RequestSaver, resp repo.ResponseSaver) {
	router := mux.NewRouter()

	handler, err := api.NewHandler(req, resp)
	if err != nil {
		log.Fatal(err)
	}

	router.HandleFunc("/requests", handler.ListRequests)
	router.HandleFunc("/requests/{id}", handler.GetRequest)
	router.HandleFunc("/repeat/{id}", handler.RepeatRequest)
	router.HandleFunc("/scan/{id}", handler.ScanRequest)
	router.HandleFunc("/requests/{id}/dump", handler.DumpRequest)

	router.HandleFunc("/responses", handler.ListResponses)
	router.HandleFunc("/responses/{id}", handler.GetResponse)
	router.HandleFunc("/requests/{id}/response", handler.GetRequestResponse)

	log.Println("Api listening at port 8000...")

	http.ListenAndServe(":8000", router)
}

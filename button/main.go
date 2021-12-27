package main

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"encoding/json"
	"net/http"
	"os"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var pool connPool

type dummyResponse struct {
	Status string
	Key    string
	TTL    time.Duration
}

type connPool struct {
	client *redis.Client
}

var (
	redisHost = os.Getenv("REDIS_HOST")
	redisPort = os.Getenv("REDIS_PORT")
	redisDB   = os.Getenv("REDIS_DB")
	ctx       = context.Background()
)

func onConnectRedisHandler(ctx context.Context, cn *redis.Conn) error {
	log.Info("New Redis Client Connection Established")
	return nil
}

// InitConnPool - Init a Connection Pool to the DB...
func InitConnPool() (*redis.Client, error) {

	redisDBID, err := strconv.Atoi(redisDB)

	if err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:       fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password:   os.Getenv("REDISCLI_AUTH"),
		DB:         redisDBID,
		MaxRetries: 5,
		OnConnect:  onConnectRedisHandler,
	})

	return client, nil

}

// HealthCheck - A *VERY* minimal route to handle ECS healthchecks!!
func HealthCheck(w http.ResponseWriter, r *http.Request) {

	log.WithFields(log.Fields{
		"Route":       r.RequestURI,
		"Remote Addr": r.RemoteAddr,
		"User-Agent":  r.Header.Get("USER-AGENT"),
	}).Info("Handling HealthCheck Request")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(&dummyResponse{Status: "OK"})
}

// ConnectToDB - A *VERY* minimal route to handle ECS <> DB connectivity
func ConnectToDB(w http.ResponseWriter, r *http.Request) {

	log.WithFields(log.Fields{
		"Route":       r.RequestURI,
		"Remote Addr": r.RemoteAddr,
		"User-Agent":  r.Header.Get("USER-AGENT"),
	}).Info("Handling DB Request")

	ttl, err := pool.client.TTL(
		ctx, "value",
	).Result()

	err = pool.client.Set(
		ctx,
		"value",
		"foo",
		time.Duration(time.Second*1),
	).Err()

	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(
			&dummyResponse{
				Status: "Hit DB",
				Key:    "value",
				TTL:    time.Duration(ttl) * 1 / time.Duration(math.Pow(10, 9)),
			})
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(&dummyResponse{Status: "NOT OK"})
	}
}

// Init runs before the main() call to save just a little bit of time.
// 	1. Set logging levels && attach logging handlers
//	2. Start Redis an Db connections
func init() {

	// Set logging config conditional on the environment - Always to STDOUT
	// and always with a specific time & msg format...
	log.SetOutput(os.Stdout)

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.0000",
	})

	// Env specific environment variables, this `AWS_EXECUTION_ENV` will exist
	// by default on AWS, and allows local mock of AWS environment....
	log.SetLevel(log.InfoLevel)
	log.SetReportCaller(false)

	client, err := InitConnPool()

	if err != nil {

		log.WithFields(log.Fields{
			"Err": err,
		}).Fatal("Cannot Connect to Redis")

	} else {
		pool.client = client
	}
}

func main() {

	// Init router and define routes for the API
	router := mux.NewRouter().StrictSlash(true)

	// HealthCheck - For determining if the container is healthy or not...
	router.Path("/health/").
		HandlerFunc(HealthCheck).
		Methods("GET")

	router.Path("/work/").
		HandlerFunc(ConnectToDB).
		Methods("GET")

	// Start service w. a graceful shutdown, lifted from mux docs: https://github.com/gorilla/mux#graceful-shutdown
	srv := &http.Server{
		Addr:         "0.0.0.0:2151",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.WithFields(log.Fields{"Error": err}).Error("API Exited")
	}

}

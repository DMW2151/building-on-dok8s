package main

import (
	"context"
	"fmt"
	"strconv"

	"encoding/json"
	"net/http"
	"os"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/mmcloughlin/geohash"
	log "github.com/sirupsen/logrus"
)

var pool connPool

type position struct {
	Lat, Lng float64
}

type response struct {
	Route, Status string
	Vehicles      []position
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

// ConnectToDB - A *VERY* minimal route to handle ECS <> DB connectivity
func getLocations(w http.ResponseWriter, r *http.Request) {

	var (
		searchSet []string
		lat, lng  float64
	)

	muxvars := mux.Vars(r)

	res, err := pool.client.Do(
		ctx,
		"SCAN",
		0,
		"MATCH",
		fmt.Sprintf("positions:%s:*", muxvars["routeid"]),
		"COUNT",
		3000,
	).Result()

	if err == nil {

		for _, m := range res.([]interface{})[1].([]interface{}) {
			searchSet = append(searchSet, m.(string))
		}

		res, err = pool.client.MGet(
			ctx, searchSet...,
		).Result()

		var vehicles = make([]position, len(searchSet))

		for i, bus := range res.([]interface{}) {

			numericHash, _ := strconv.ParseUint(bus.(string), 10, 64)

			lat, lng = geohash.DecodeIntWithPrecision(numericHash, 64)
			vehicles[i] = position{lat, lng}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(
			&response{
				Route:    muxvars["routeid"],
				Status:   "OK",
				Vehicles: vehicles,
			},
		)

		return

	} else if err.Error() == "redis: nil" {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(
			&response{
				Route:    muxvars["routeid"],
				Status:   "OK",
				Vehicles: nil,
			})
	} else {

		log.WithFields(log.Fields{
			"Route":       r.RequestURI,
			"Remote Addr": r.RemoteAddr,
			"User-Agent":  r.Header.Get("USER-AGENT"),
			"Error":       err.Error(),
		}).Error("Handling Location Request")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(
			&response{
				Route:    muxvars["routeid"],
				Status: "NOT OK"
				Vehicles: nil,
			}
		)
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

	router.Path("/locations/{routeid:[A-Za-z0-9]+}").
		HandlerFunc(getLocations).
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

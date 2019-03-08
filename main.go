package main

import (
  "encoding/json"
  "fmt"
  "log"
  "net/http"
  "os"
  "strconv"
  "time"

  "github.com/joho/godotenv"
  "github.com/julienschmidt/httprouter"
  "github.com/go-redis/redis"
)

var listen string
var redisHost string

type RawClient struct {
  UserId string `json:"userid"`
  ClientId string `json:"clientid"`
}

type Ping struct {
  Time string `json:"time"`
  Status string `json:"status"`
}

var connections map[RawClient][]chan []byte
var redisClient *redis.Client

func main() {
  // Load .env
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }
  listen = os.Getenv("LISTEN")
  redisHost = os.Getenv("REDIS")

  connections = make(map[RawClient][]chan []byte)

  // Redis
  redisClient = redis.NewClient(&redis.Options{
    Addr: redisHost,
    Password: "",
    DB: 0,
  })

  // Routes
	router := httprouter.New()
  router.GET("/subscribe/:userid/client/:clientid", Subscribe)
  router.POST("/ping", PostTime)

  // Start server
  log.Printf("starting server on %s", listen)
	log.Fatal(http.ListenAndServe(listen, router))
}

func Subscribe(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
  flusher, ok := w.(http.Flusher)
  if !ok {
    http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
    return
  }

  w.Header().Set("Content-Type", "text/event-stream")
  w.Header().Set("Cache-Control", "no-cache")
  w.Header().Set("Connection", "keep-alive")

  client := RawClient {
    UserId: p.ByName("userid"),
    ClientId: p.ByName("clintid"),
  }
  recv := make(chan []byte)
  connections[client] = append(connections[client], recv);

  // Refresh connection periodically
  resClosed := w.(http.CloseNotifier).CloseNotify()
  ticker := time.NewTicker(25 * time.Second)

  // Push cached value (if it exists) to the connection
  cachedTime, err1 := redisClient.HGet(client.UserId + client.ClientId, "time").Result()
  cachedStatus, err2 := redisClient.HGet(client.UserId + client.ClientId, "status").Result()
  if err1 == nil && err2 == nil {
    ping := Ping {
      Time: cachedTime,
      Status: cachedStatus,
    }
    pingBytes, err := json.Marshal(&ping)
    if err == nil {
      fmt.Fprintf(w, "data: %s\n\n", pingBytes)
      flusher.Flush()
    }
  }

  for {
    select {
      case msg := <- recv:
        fmt.Fprintf(w, "data: %s\n\n", msg)
        flusher.Flush()
      case <- ticker.C:
        w.Write([]byte(":\n\n"))
      case <- resClosed:
        ticker.Stop()
        delete(connections, client)
        return
    }
  }
}

type PostTimeRequest struct {
  Status string `json:"status"`
}
func PostTime(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
  ua := r.Header.Get("X-User-Claim")
  if ua == "" {
    http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
  }

  var client RawClient
  err := json.Unmarshal([]byte(ua), &client)

  if err != nil {
    http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
  }

  decoder := json.NewDecoder(r.Body)
  var ptRequest PostTimeRequest
  err = decoder.Decode(&ptRequest)
  if err != nil {
    http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
  }

  ping := Ping {
    Time: strconv.FormatInt(time.Now().UTC().Unix(), 10), // UTC Epoch time,
    Status: ptRequest.Status,
  }

  key := client.UserId + client.ClientId
  _ = redisClient.HSet(key, "time", []byte(ping.Time))
  _ = redisClient.HSet(key, "status", []byte(ping.Status))

  pingBytes, err := json.Marshal(&ping)
  if err != nil {
    http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
  }

  for _, connection := range connections[client] {
    connection <- pingBytes
  }

  w.WriteHeader(http.StatusOK)
}

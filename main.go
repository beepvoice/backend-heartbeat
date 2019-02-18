package main

import (
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
  UserId string
  ClientId string
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
  router.POST("/ping/:userid/client/:clientid", PostTime)

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
  cachedTime, err := redisClient.Get(client.UserId + client.ClientId).Result()
  if err == nil {
    fmt.Fprintf(w, "data: %s\n\n", cachedTime)
    flusher.Flush()
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

// TODO: Take client data from token
func PostTime(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
  client := RawClient {
    UserId: p.ByName("userid"),
    ClientId: p.ByName("clientid"),
  }

  time := []byte(strconv.FormatInt(time.Now().UTC().Unix(), 10)) // UTC Epoch Time in []byte
  key := client.UserId + client.ClientId
  _ = redisClient.Set(key, time, 0)

  for _, connection := range connections[client] {
    connection <- time
  }

  w.WriteHeader(http.StatusOK)
}

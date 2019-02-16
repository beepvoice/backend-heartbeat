package main

import (
  "flag"
  "fmt"
  "log"
  "net/http"
  "strconv"
  "time"

  "github.com/julienschmidt/httprouter"
)

var listen string

type RawClient struct {
  UserId string
  ClientId string
}

var connections map[RawClient][]chan []byte

func main() {
  // Parse flags
  flag.StringVar(&listen, "listen", ":8080", "host and port to listen on")
  flag.Parse()

  connections = make(map[RawClient][]chan []byte)

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

  time := time.Now().UTC().Unix()

  for _, connection := range connections[client] {
    connection <- []byte(strconv.FormatInt(time, 10))
  }

  w.WriteHeader(http.StatusOK)
}

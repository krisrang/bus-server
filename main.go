package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	mux "github.com/gorilla/mux"
)

var url = "http://soiduplaan.tallinn.ee/gps.txt"
var port int
var markers map[string]map[string]string

func init() {
	flag.IntVar(&port, "port", 5000, "port to run bus-server on")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %v [options]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	markers = make(map[string]map[string]string)
	portString := strconv.Itoa(port)

	stop := make(chan bool)
	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				update()
			case <-stop:
				return
			}
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/gps", gps)
	http.Handle("/", &CorsServer{r})
	fmt.Printf("bus-server listening on port %s\n", portString)
	err := http.ListenAndServe(":"+portString, nil)
	if err != nil {
		panic(err)
	}
}

type CorsServer struct {
	r *mux.Router
}

func (s *CorsServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if origin := req.Header.Get("Origin"); origin != "" {
		rw.Header().Set("Access-Control-Allow-Origin", origin)
		rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		rw.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}
	// Stop here if its Preflighted OPTIONS request
	if req.Method == "OPTIONS" {
		return
	}
	// Lets Gorilla work
	s.r.ServeHTTP(rw, req)
}

func gps(res http.ResponseWriter, req *http.Request) {
	jsonString, err := json.Marshal(markers)
	if err != nil {
		fmt.Fprintln(res, err)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(res, "%s", jsonString)
}

func update() {
	gps, _ := download(url)
	lines := strings.Split(string(gps[:]), "\n")
	for i := 0; i < len(lines); i++ {
		item := strings.Split(lines[i], ",")

		if len(item) < 7 {
			continue
		}

		marker := make(map[string]string)
		marker["id"] = item[6]
		marker["type"] = item[0]
		marker["number"] = item[1]
		marker["long"] = item[2]
		marker["lat"] = item[3]
		marker["dir"] = item[5]
		markers[marker["id"]] = marker
	}
	log.Println("updated gps")
}

func download(uri string) ([]byte, error) {
	res, err := http.Get(uri)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	return d, err
}

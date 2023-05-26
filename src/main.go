package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/carlmjohnson/gateway"
)

func main() {
	port := flag.Int("port", -1, "specify a port to use http rather than AWS Lambda")
	flag.Parse()
	listener := gateway.ListenAndServe
	portStr := ""
	if *port != -1 {
		portStr = fmt.Sprintf(":%d", *port)
		listener = http.ListenAndServe
		http.Handle("/", http.FileServer(http.Dir("./public")))
	}

	http.HandleFunc("/api/farmers/find", func(w http.ResponseWriter, r *http.Request) {
		sLongitude := r.URL.Query().Get("location_longitude")
		var longitude float64
		var err error
		if len(sLongitude) > 0 {
			longitude, err = strconv.ParseFloat(sLongitude, 64)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			w.Write([]byte("The parameter 'location_longitude' is required."))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sLatitude := r.URL.Query().Get("location_latitude")
		var latitude float64
		if len(sLatitude) > 0 {
			latitude, err = strconv.ParseFloat(sLatitude, 64)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			w.Write([]byte("The parameter 'location_latitude' is required."))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sMaxDistance_km := r.URL.Query().Get("maxDistance_km")
		var maxDistance_km float64
		if len(sMaxDistance_km) > 0 {
			maxDistance_km, err = strconv.ParseFloat(sMaxDistance_km, 64)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			maxDistance_km = 50
		}
		sGroceryTypes := r.URL.Query().Get("filter_groceryTypes")
		groceryTypes := make([]string, 0)
		for _, groceryType := range strings.Split(sGroceryTypes, ",") {
			if len(groceryType) > 0 {
				groceryTypes = append(groceryTypes, groceryType)
			}
		}
		// openingHours_ISO8601 := r.URL.Query().Get("filter_openingHours_ISO8601")

		farmers, err := getFramersNearBy(
			geoLocation{Longitude: longitude, Latitude: latitude},
			maxDistance_km,
			groceryTypes,
			// openingHours_ISO8601,
		)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		b, err := json.Marshal(farmers)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	})
	log.Fatal(listener(portStr, nil))
}

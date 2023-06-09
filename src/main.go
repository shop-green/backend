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
	"github.com/gorilla/mux"
)

func main() {
	port := flag.Int("port", -1, "specify a port to use http rather than AWS Lambda")
	flag.Parse()
	r := mux.NewRouter()
	listener := gateway.ListenAndServe
	portStr := ""
	if *port != -1 {
		portStr = fmt.Sprintf(":%d", *port)
		listener = http.ListenAndServe
		r.Handle("/", http.FileServer(http.Dir("./public")))
	}

	r.HandleFunc("/api/farmers/find", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

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
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("The parameter 'location_longitude' is required."))
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
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("The parameter 'location_latitude' is required."))
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
		sFeatures := r.URL.Query().Get("filter_features")
		features := make([]string, 0)
		for _, feature := range strings.Split(sFeatures, ",") {
			if len(feature) > 0 {
				features = append(features, feature)
			}
		}

		farmers, err := getFarmersNearBy(
			geoLocation{Longitude: longitude, Latitude: latitude},
			maxDistance_km,
			groceryTypes,
			features,
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

	r.HandleFunc("/api/farmers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// deserialize farmer from request body
		var farmer farmer
		err = json.Unmarshal(body, &farmer)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid JSON"))
			return
		}

		// add farmer
		farmer, err = addFarmer(farmer)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		b, err := json.Marshal(farmer)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	})

	r.HandleFunc("/api/farmers/{id}/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method != "POST" && r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// get farmer id from path
		farmerId := strings.TrimPrefix(r.URL.Path, "/api/farmers/")
		farmerId = strings.TrimSuffix(farmerId, "/products")
		_, err := fromJsonFarmerId(farmerId)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid farmer id"))
			return
		}

		if r.Method == "GET" {
			products, err := getProductsByFarmer(farmerId)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			b, err := json.Marshal(products)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "public, max-age=300")
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		} else if r.Method == "POST" {
			defer r.Body.Close()

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// deserialize products from request body
			var products []product
			err = json.Unmarshal(body, &products)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid JSON"))
				return
			}

			// add product
			products, err = addProducts(farmerId, products)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			b, err := json.Marshal(products)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	})

	log.Fatal(listener(portStr, r))
}

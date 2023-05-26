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

	"github.com/carlmjohnson/gateway"
)

// farmers slice to seed record farmer data.
var farmers = []farmer{
	{ID: "f23098490", Name: "The local Farm <3", Rating: 4.6, GroceryTypes: []string{"Strawberry", "Potato"}, OpeningHoursByDayOfWeekSecondsFromStartOfDay: map[string][][]int32{"Monday": {{9 * 3600, 17 * 3600}}, "Tuesday": {{9 * 3600, 17 * 3600}}, "Wednesday": {{9 * 3600, 17 * 3600}}, "Thursday": {{9 * 3600, 17 * 3600}}, "Friday": {{9 * 3600, 17 * 3600}}, "Saturday": {{9 * 3600, 17 * 3600}}, "Sunday": {{9 * 3600, 17 * 3600}}}, TitleImage: "/img/2l092834lskhsieo.svg"},
	{ID: "f09384053", Name: "Jeru", Rating: 4.3},
	{ID: "f03498234", Name: "Sarah Vaughan and Clifford Brown", Rating: 4.6},
}

func getFramerIdsAndDistancesNearByFromKinetica(point geoLocation, maxDistance_km float64) (map[string]float64, error) {
	url := os.Getenv("KINETICA_BASE_URL") + "/execute/sql"
	method := "GET"

	query := fmt.Sprintf(`{
		"statement": "SELECT ID, GEODIST(stores.longitude, stores.latitude, %.9f, %.9f) AS distance_m FROM stores WHERE GEODIST(stores.longitude, stores.latitude, %.9f, %.9f) < %.9f;",
		"offset": 0,
		"limit": 100,
		"encoding": "json"
	}`, point.Longitude, point.Latitude, point.Longitude, point.Latitude, maxDistance_km*1000)
	payload := strings.NewReader(query)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", os.Getenv("KINETICA_AUTHORIZATION"))

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	resp, err := parseBodyAsKineticaResponse(body)
	if err != nil {
		return nil, err
	}
	sqlResp, err := parseExecuteSqlResponse(resp.DataStr)
	if err != nil {
		return nil, err
	}
	rows, err := parseJsonEncodedResponseAsListOfMaps(sqlResp.JsonEncodedResponse)
	if err != nil {
		return nil, err
	}

	idsAndDistances := make(map[string]float64)
	for _, row := range rows {
		idsAndDistances[row["ID"].(string)] = row["distance_m"].(float64)
	}
	return idsAndDistances, nil
}

func getFramersNearBy(
	point geoLocation,
	maxDistance_km float64,
	groceryTypes []string,
	// openingHours time.Time,
) ([]farmer, error) {
	idsAndDistances, err := getFramerIdsAndDistancesNearByFromKinetica(point, maxDistance_km)
	if err != nil {
		return nil, err
	}
	fmt.Println(idsAndDistances)
	return farmers, nil
}

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
		groceryTypes := strings.Split(sGroceryTypes, ",")
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

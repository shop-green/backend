package main

import (
	"context"
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

	"github.com/carlmjohnson/gateway"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/maps"
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
		"statement": "SELECT id, GEODIST(farmers.longitude, farmers.latitude, %.9f, %.9f) AS distance_m FROM farmers WHERE GEODIST(farmers.longitude, farmers.latitude, %.9f, %.9f) < %.9f;",
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
	if resp.Status != "OK" {
		return nil, fmt.Errorf("Kinetica response status is %s (expected OK): %s", resp.Status, resp.Message)
	}
	if resp.DataType != "execute_sql_response" {
		return nil, fmt.Errorf("Kinetica response data_type is %s (expected execute_sql_response): %s", resp.DataType, resp.Message)
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
		idsAndDistances[row["id"].(string)] = row["distance_m"].(float64)
	}
	return idsAndDistances, nil
}

func getFarmersByFiltersFromMongo(
	ids []string,
	groceryTypes []string,
) ([]farmer, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGODB_CONNECTION_STRING")))
	if err != nil {
		return nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(ctx)

	coll := client.Database("shopGreenDB").Collection("farmers")

	// convert ids to bson object ids
	objectIds := make([]primitive.ObjectID, 0)
	for _, id := range ids {
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		objectIds = append(objectIds, objectId)
	}
	// a filter for mongodb query that checks if the id is in the ids array and the groceryTypes array is a subset of the groceryTypes array in the document
	var filter bson.D
	if len(groceryTypes) <= 0 {
		filter = bson.D{{"_id", bson.D{{"$in", objectIds}}}}
	} else {
		filter = bson.D{
			{"$and",
				bson.A{
					bson.D{{"_id", bson.D{{"$in", objectIds}}}},
					bson.D{{"groceryTypes", bson.D{{"$all", groceryTypes}}}},
				}},
		}
	}
	// sort := bson.D{{"date_ordered", 1}}
	opts := options.Find() //.SetSort(sort)

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	var results []farmer = make([]farmer, 0)
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
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
	farmers, err := getFarmersByFiltersFromMongo(maps.Keys(idsAndDistances), groceryTypes)
	if err != nil {
		return nil, err
	}
	for i, farmer := range farmers {
		farmers[i].ID = "f" + farmer.MongoDbID
		farmers[i].Distance_km = idsAndDistances[farmer.MongoDbID] / 1000
	}
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

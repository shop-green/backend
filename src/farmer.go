package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/maps"
)

type farmer struct {
	MongoDbID                                    string               `bson:"_id" json:"-"`
	ID                                           string               `bson:"-" json:"id"`
	Name                                         string               `bson:"name" json:"name"`
	Rating                                       float32              `bson:"rating" json:"rating"`
	GroceryTypes                                 []string             `bson:"groceryTypes" json:"groceryTypes"`
	TitleImage                                   string               `bson:"titleImage" json:"titleImage"`
	Address                                      address              `bson:"address" json:"address"`
	Location                                     geoLocation          `bson:"location" json:"location"`
	Features                                     []string             `bson:"features" json:"features"`
	OpeningHoursByDayOfWeekSecondsFromStartOfDay map[string][][]int32 `bson:"openingHoursByDayOfWeek_secondsFromStartOfDay" json:"openingHoursByDayOfWeek_secondsFromStartOfDay"`
	Distance_km                                  float64              `bson:"-" json:"distance_km"`
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

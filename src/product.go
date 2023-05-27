package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type price struct {
	Value   float32 `bson:"value,omitempty" json:"value,omitempty"`
	PerUnit string  `bson:"perUnit,omitempty" json:"perUnit,omitempty"`
}

type product struct {
	MongoDbID       primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	ID              string             `bson:"-" json:"id,omitempty"`
	MongoDbFarmerID primitive.ObjectID `bson:"farmerId,omitempty" json:"-"`
	FarmerID        string             `bson:"-" json:"farmerId,omitempty"`
	Name            string             `bson:"name,omitempty" json:"name,omitempty"`
	GroceryType     string             `bson:"groceryType,omitempty" json:"groceryType,omitempty"`
	Description     string             `bson:"description,omitempty" json:"description,omitempty"`
	Price           price              `bson:"price,omitempty" json:"price,omitempty"`
	TitleImage      string             `bson:"titleImage,omitempty" json:"titleImage,omitempty"`
}

func toJsonProductId(id primitive.ObjectID) string {
	return "p-" + id.Hex()
}

func fromJsonProductId(id string) (primitive.ObjectID, error) {
	// if id does not start with "p-", then it is not a product id
	if !strings.HasPrefix(id, "p-") {
		return primitive.ObjectID{}, fmt.Errorf("Invalid id: %s", id)
	}
	return primitive.ObjectIDFromHex(id[2:])
}

func getProductsByFarmer(farmerId string) ([]product, error) {
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

	coll := client.Database("shopGreenDB").Collection("products")

	// convert farmerId to bson object ids
	farmerObjectId, err := fromJsonFarmerId(farmerId)
	if err != nil {
		return nil, err
	}
	filter := bson.D{{"farmerId", bson.D{{"$eq", farmerObjectId}}}}
	// sort := bson.D{{"date_ordered", 1}}
	opts := options.Find() //.SetSort(sort)

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	var results []product = make([]product, 0)
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	for i, result := range results {
		results[i].ID = toJsonProductId(result.MongoDbID)
	}

	return results, nil
}

func addProducts(farmerId string, products []product) ([]product, error) {
	// convert farmerId to bson object ids
	farmerObjectId, err := fromJsonFarmerId(farmerId)
	if err != nil {
		return nil, err
	}

	for i := range products {
		products[i].MongoDbID = primitive.ObjectID{}
		products[i].MongoDbFarmerID = farmerObjectId
	}

	// Connect to MongoDB
	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGODB_CONNECTION_STRING")))
	if err != nil {
		return products, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return products, err
	}
	defer client.Disconnect(ctx)

	// Insert products
	coll := client.Database("shopGreenDB").Collection("products")
	documents := make([]interface{}, 0)
	for _, product := range products {
		documents = append(documents, product)
	}
	result, err := coll.InsertMany(ctx, documents)
	if err != nil {
		return products, err
	}
	if len(result.InsertedIDs) != len(products) {
		return products, fmt.Errorf("Expected %d inserted ids, got %d", len(products), len(result.InsertedIDs))
	}
	for i := range products {
		products[i].MongoDbID = result.InsertedIDs[i].(primitive.ObjectID)
		products[i].ID = toJsonProductId(products[i].MongoDbID)
		products[i].FarmerID = toJsonFarmerId(products[i].MongoDbFarmerID)
	}

	// mongo db update farmer's grocery types to include the new product's grocery types
	groceryTypes := make([]string, 0)
	for _, product := range products {
		groceryTypes = append(groceryTypes, product.GroceryType)
	}
	filter := bson.D{{"_id", bson.D{{"$eq", farmerObjectId}}}}
	update := bson.D{{"$addToSet", bson.D{{"groceryTypes", bson.D{{"$each", groceryTypes}}}}}}
	collFarmers := client.Database("shopGreenDB").Collection("farmers")
	_, err = collFarmers.UpdateOne(ctx, filter, update)
	if err != nil {
		return products, err
	}

	return products, nil
}

package main

type address struct {
	Street  string `bson:"street" json:"street"`
	City    string `bson:"city" json:"city"`
	ZipCode string `bson:"zipCode" json:"zipCode"`
	Country string `bson:"country" json:"country"`
}

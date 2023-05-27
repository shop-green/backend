package main

type address struct {
	Street  string `bson:"street,omitempty" json:"street,omitempty"`
	City    string `bson:"city,omitempty" json:"city,omitempty"`
	ZipCode string `bson:"zipCode,omitempty" json:"zipCode,omitempty"`
	Country string `bson:"country,omitempty" json:"country,omitempty"`
}

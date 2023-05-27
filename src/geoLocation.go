package main

type geoLocation struct {
	Longitude float64 `bson:"longitude,omitempty" json:"longitude,omitempty"`
	Latitude  float64 `bson:"latitude,omitempty" json:"latitude,omitempty"`
}

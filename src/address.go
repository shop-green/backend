package main

type address struct {
	Street  string `json:"street"`
	Number  string `json:"number"`
	City    string `json:"city"`
	ZipCode string `json:"zipCode"`
	Country string `json:"country"`
}

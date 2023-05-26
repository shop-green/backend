package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/carlmjohnson/gateway"
)

// farmer represents data about a farmer.
type farmer struct {
	ID                                           string               `json:"id"`
	Name                                         string               `json:"name"`
	Rating                                       float64              `json:"rating"`
	GroceryTypes                                 []string             `json:"groceryTypes"`
	OpeningHoursByDayOfWeekSecondsFromStartOfDay map[string][][]int32 `json:"openingHoursByDayOfWeek_secondsFromStartOfDay"`
	TitleImage                                   string               `json:"titleImage"`
}

// farmers slice to seed record farmer data.
var farmers = []farmer{
	{ID: "f23098490", Name: "The local Farm <3", Rating: 4.6, GroceryTypes: []string{"Strawberry", "Potato"}, OpeningHoursByDayOfWeekSecondsFromStartOfDay: map[string][][]int32{"Monday": {{9 * 3600, 17 * 3600}}, "Tuesday": {{9 * 3600, 17 * 3600}}, "Wednesday": {{9 * 3600, 17 * 3600}}, "Thursday": {{9 * 3600, 17 * 3600}}, "Friday": {{9 * 3600, 17 * 3600}}, "Saturday": {{9 * 3600, 17 * 3600}}, "Sunday": {{9 * 3600, 17 * 3600}}}, TitleImage: "/img/2l092834lskhsieo.svg"},
	{ID: "f09384053", Name: "Jeru", Rating: 4.3},
	{ID: "f03498234", Name: "Sarah Vaughan and Clifford Brown", Rating: 4.6},
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
		b, err := json.Marshal(farmers)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	})
	log.Fatal(listener(portStr, nil))
}

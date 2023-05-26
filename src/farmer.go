package main

type farmer struct {
	ID                                           string               `json:"id"`
	Name                                         string               `json:"name"`
	Rating                                       float32              `json:"rating"`
	GroceryTypes                                 []string             `json:"groceryTypes"`
	TitleImage                                   string               `json:"titleImage"`
	Address                                      address              `json:"address"`
	Location                                     geoLocation          `json:"location"`
	Features                                     []string             `json:"features"`
	OpeningHoursByDayOfWeekSecondsFromStartOfDay map[string][][]int32 `json:"openingHoursByDayOfWeek_secondsFromStartOfDay"`
	Distance_km                                  float64              `json:"distance_km"`
}

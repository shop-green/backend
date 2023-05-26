package main

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

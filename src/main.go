package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	{ID: "f23098490", Name: "The local Farm <3", Rating: 4.6, GroceryTypes: []string{"Strawberry", "Potato"}, OpeningHoursByDayOfWeekSecondsFromStartOfDay: map[string][][]int32{"Monday": {{9 * 3600, 17 * 3600}}, "Tuesday": {{9 * 3600, 17 * 3600}}, "Wednesday": {{9 * 3600, 17 * 3600}}, "Thursday": {{9 * 3600, 17 * 3600}}, "Friday": {{9 * 3600, 17 * 3600}}, "Saturday": {{9 * 3600, 17 * 3600}}, "Sunday": {{9 * 3600, 17 * 3600}}}, TitleImage: "https://localhost/img/2l092834lskhsieo.svg"},
	{ID: "f09384053", Name: "Jeru", Rating: 4.3},
	{ID: "f03498234", Name: "Sarah Vaughan and Clifford Brown", Rating: 4.6},
}

// getFarmers responds with the list of all farmers as JSON.
func getFarmers(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, farmers)
}

func main() {
	router := gin.Default()
	router.GET("/api/farmers/find", getFarmers)

	router.Run("localhost:8080")
}

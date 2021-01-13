package web

import (
	"net/http"

	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/dingxizheng/sms-bot/utils"
	"github.com/gosimple/slug"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GenerateSiteMapFile(w http.ResponseWriter, r *http.Request) {
	matchStage := bson.M{"status": "online", "country": bson.M{"$ne": ""}}
	groupStage := bson.M{"_id": "$country", "count": bson.M{"$sum": 1}}
	sortStage := bson.M{"count": -1}

	coll := db.Collection("numbers")

	pipeline := mongo.Pipeline{
		{{"$match", matchStage}},
		{{"$group", groupStage}},
		{{"$sort", sortStage}},
	}

	cursor, _ := coll.Aggregate(db.DefaultCtx(), pipeline)

	var countries = make([]models.SMSCountry, 0)
	for cursor.Next(db.DefaultCtx()) {
		country := models.SMSCountry{}
		cursor.Decode(&country)
		country.CountryName = utils.FindCountryName(country.Country)
		country.CountrySlug = slug.Make(country.CountryName)
		countries = append(countries, country)
	}

	cursor, _ = coll.Find(db.DefaultCtx(), bson.M{"status": "online"})

	var numbers = make([]models.PhoneNumber, 0)
	for cursor.Next(db.DefaultCtx()) {
		num := models.PhoneNumber{}
		cursor.Decode(&num)
		numbers = append(numbers, num)
	}

	w.Header().Add("Content-Type", "application/xml")
	rnd.Template(w, 200, []string{"templates/sitemap.xml"}, H{"numbers": numbers, "countries": countries})
}

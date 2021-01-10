package web

import (
	"net/http"

	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func MountIndexController(router *gin.Engine) {
	router.GET("/.sitemap.xml", GenerateSiteMapFile)

	router.GET("/privacy", func(c *gin.Context) {
		c.HTML(
			http.StatusOK,
			"privacy.html",
			gin.H{
				"title": "Web",
				"url":   "./web.json",
			},
		)
	})
}

func GenerateSiteMapFile(c *gin.Context) {
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
		country.CountryName = findCountryName(country.Country)
		countries = append(countries, country)
	}

	cursor, _ = coll.Find(db.DefaultCtx(), bson.M{"status": "online"})

	var numbers = make([]models.PhoneNumber, 0)
	for cursor.Next(db.DefaultCtx()) {
		num := models.PhoneNumber{}
		cursor.Decode(&num)
		num.CountryName = findCountryName(num.Country)
		numbers = append(numbers, num)
	}

	c.Header("Content-Type", "application/xml")
	c.HTML(200, "sitemap.xml", gin.H{"numbers": numbers, "countries": countries})
}

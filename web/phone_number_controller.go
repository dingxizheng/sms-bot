package web

import (
	"fmt"
	"log"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/dingxizheng/sms-bot/utils"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const pageSize = 30

func requestURL(url *url.URL) string {
	return fmt.Sprintf("https://freesms-online.com%v", url.String())
}

func MountPhoneNumberController(router *gin.Engine) {

	router.GET("/", FindNumbers)
	router.GET("/home", FindNumbers)
	router.GET("/phone-numbers", FindNumbers)
	router.GET("/phone-numbers/:country/:page", FindNumbers)
	router.GET("/index", FindNumbers)

	router.GET("/free-sms-messages/:provider/:providerId", FindMessages)
	router.GET("/messages", FindMessages)

	router.GET("/countries", FindCountries)
}

// FindNumbers - render numbers
func FindNumbers(c *gin.Context) {
	page := c.Param("page")
	country := c.Param("country")
	countryName := country

	if len(country) == 0 {
		country = "all-countries"
	} else {
		countryName = utils.FindCountryName(country)
	}

	filters := bson.M{"status": "online"}
	if country != "all-countries" {
		filters["country"] = strings.ToUpper(country)
	}

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum <= 0 {
		pageNum = 1
	}

	coll := db.Collection("numbers")
	totalNums, _ := coll.CountDocuments(db.DefaultCtx(), filters)
	totalPages := int(math.Ceil(float64(totalNums) / float64(pageSize)))
	cursor, err := coll.Find(db.DefaultCtx(), filters, options.Find().SetLimit(pageSize).SetSkip(int64((pageNum-1)*pageSize)))

	if err != nil {
		log.Printf("Failed to fetch phone numbers, error: %+v", err)
		c.HTML(500, "numbers.html", gin.H{"error": "Oops, something went wrong, please try again later"})
		return
	}

	var numbers = make([]models.PhoneNumber, 0)
	for cursor.Next(db.DefaultCtx()) {
		num := models.PhoneNumber{}
		cursor.Decode(&num)
		num.CountryName = utils.FindCountryName(num.Country)
		numbers = append(numbers, num)
	}

	if err != nil {
		log.Printf("Failed to decode phone numbers, error: %+v", err)
		c.HTML(500, "numbers.html", gin.H{"error": "Oops, something went wrong, please try again later"})
		return
	}

	pages := utils.Pagination(pageNum, totalPages)
	for idx := range pages {
		pages[idx].URL = fmt.Sprintf("/phone-numbers/%s/%d", country, pages[idx].Current)
		if pageNum == pages[idx].Current {
			pages[idx].Active = true
		} else {
			pages[idx].Active = false
		}
	}
	previousURL := fmt.Sprintf("/phone-numbers/%s/%d", country, pageNum-1)
	nextURL := fmt.Sprintf("/phone-numbers/%s/%d", country, pageNum+1)

	c.HTML(200, "numbers.html", gin.H{
		"numbers":     numbers,
		"hasPrevious": pageNum > 1,
		"hasNext":     pageNum < totalPages,
		"previousURL": previousURL,
		"nextURL":     nextURL,
		"pages":       pages,
		"hasCountry":  len(c.Param("country")) != 0,
		"countryName": countryName,
		"metaData": gin.H{
			"pageURL": requestURL(c.Request.URL),
		},
	})
}

// FindMessages - returns the most recent messages of the given number
func FindMessages(c *gin.Context) {
	provider := c.Param("provider")
	providerID := c.Param("providerId")
	autotriggering, _ := c.Cookie("autotriggering")
	coll := db.Collection("numbers")
	var phoneNumber models.PhoneNumber
	err := coll.FindOne(db.DefaultCtx(), bson.M{"provider": provider, "provider_id": providerID}).Decode(&phoneNumber)

	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{"status", "online"}}}},
		{{"$sample", bson.D{{"size", 1}}}},
	}

	var randomNum models.PhoneNumber
	aggregateRs, _ := coll.Aggregate(db.DefaultCtx(), pipeline)
	for aggregateRs.Next(db.DefaultCtx()) {
		aggregateRs.Decode(&randomNum)
		break
	}

	if err != nil {
		log.Printf("Failed to fetch phone numbers, error: %+v", err)
		c.HTML(500, "messages.html", gin.H{"error": "Oops, something went wrong, please try again later"})
		return
	}

	messages := phoneNumber.Messages
	shouldScheduleJob := false

	if phoneNumber.LastReadAt.Equal(time.Time{}) {
		if phoneNumber.NextReadAt.Before(time.Now()) {
			shouldScheduleJob = true
		}
	} else if phoneNumber.NextReadAt.Equal(time.Time{}) || phoneNumber.NextReadAt.Before(time.Now()) {
		shouldScheduleJob = true
	}

	if shouldScheduleJob && len(autotriggering) == 0 {
		nextRunAt := time.Now().Add(5 * time.Second)
		phoneNumber.NextReadAt = nextRunAt
		log.Printf("Plan to read messages for number(%v): %v at %v", phoneNumber.Provider, phoneNumber.RawNumber, nextRunAt)
		coll.UpdateOne(db.DefaultCtx(), bson.M{"_id": phoneNumber.ID}, bson.M{"$set": bson.M{"next_read_at": nextRunAt}})
	} else {
		log.Printf("Aleady planed to read messages for number %v at %v", phoneNumber.RawNumber, phoneNumber.NextReadAt)
	}

	if err != nil {
		log.Printf("Failed to decode phone numbers, error: %+v", err)
		c.HTML(500, "messages.html", gin.H{"error": "Oops, something went wrong, please try again later"})
		return
	}

	// Create channel
	channelID := uuid.NewV4()
	providerNumber := phoneNumber.Provider + "|" + phoneNumber.ProviderID
	_, exists := NumberChannels[channelID.String()]
	if !exists {
		NumberChannels[channelID.String()] = &WebClient{
			Channel: make(chan int),
			Number:  providerNumber,
		}
	}

	c.HTML(200, "messages.html", gin.H{
		"messages":       messages,
		"countryName":    utils.FindCountryName(phoneNumber.Country),
		"randomNumber":   randomNum,
		"number":         phoneNumber,
		"nextReadAt":     fmt.Sprintf("%d000", phoneNumber.NextReadAt.Add(4*time.Second).Unix()),
		"channelID":      channelID.String(),
		"providerNumber": providerNumber,
		"metaData": gin.H{
			"pageURL": requestURL(c.Request.URL),
		},
	})
}

// FindCountries - show all available countries
func FindCountries(c *gin.Context) {
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
		countries = append(countries, country)
	}

	c.HTML(200, "countries.html", gin.H{
		"countries":      countries,
		"countriesCount": len(countries),
		"metaData": gin.H{
			"pageURL": requestURL(c.Request.URL),
		},
	})
}

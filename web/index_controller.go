package web

import (
	"log"
	"time"

	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MountIndexController(router *gin.Engine) {
	router.GET("/", FindNumbers)
	router.GET("/numbers", FindNumbers)
	router.GET("/home", FindNumbers)
	router.GET("/index", FindNumbers)

	router.GET("/free-sms-messages", FindMessages)
	router.GET("/messages", FindMessages)
}

func FindNumbers(c *gin.Context) {
	coll := db.Collection("numbers")
	cursor, err := coll.Find(db.DefaultCtx(), bson.M{"status": "online"}, options.Find().SetLimit(500).SetSkip(0))

	if err != nil {
		log.Printf("Failed to fetch phone numbers, error: %+v", err)
		c.HTML(500, "index.html", gin.H{"error": "Oops, something went wrong, please try again later"})
		return
	}

	var numbers = make([]models.PhoneNumber, 0)
	for cursor.Next(db.DefaultCtx()) {
		num := models.PhoneNumber{}
		cursor.Decode(&num)
		numbers = append(numbers, num)
	}

	if err != nil {
		log.Printf("Failed to decode phone numbers, error: %+v", err)
		c.HTML(500, "index.html", gin.H{"error": "Oops, something went wrong, please try again later"})
		return
	}

	c.HTML(200, "index.html", gin.H{"numbers": numbers})
}

// FindMessages - returns the most recent messages of the given number
func FindMessages(c *gin.Context) {
	numberID := c.Query("id")
	id, _ := primitive.ObjectIDFromHex(numberID)
	coll := db.Collection("numbers")
	var phoneNumber models.PhoneNumber
	err := coll.FindOne(db.DefaultCtx(), bson.M{"_id": id}).Decode(&phoneNumber)

	if err != nil {
		log.Printf("Failed to fetch phone numbers, error: %+v", err)
		c.HTML(500, "messages.html", gin.H{"error": "Oops, something went wrong, please try again later"})
		return
	}

	messages := phoneNumber.Messages
	shouldScheduleJob := false

	if phoneNumber.LastReadAt.Equal(time.Time{}) {
		if !phoneNumber.NextReadAt.After(time.Now()) {
			shouldScheduleJob = true
		}
	} else if !phoneNumber.NextReadAt.Equal(time.Time{}) && !phoneNumber.NextReadAt.After(time.Now()) {
		shouldScheduleJob = true
	} else if phoneNumber.NextReadAt.Equal(time.Time{}) && phoneNumber.LastReadAt.Add(10*time.Second).Before(time.Now()) {
		shouldScheduleJob = true
	}

	if shouldScheduleJob {
		nextRunAt := time.Now().Add(10 * time.Second)
		log.Printf("Schedule read message job for number %v at %v", phoneNumber.RawNumber, nextRunAt)
		coll.UpdateOne(db.DefaultCtx(), bson.M{"_id": id}, bson.M{"$set": bson.M{"next_read_at": nextRunAt}})
	}

	if err != nil {
		log.Printf("Failed to decode phone numbers, error: %+v", err)
		c.HTML(500, "messages.html", gin.H{"error": "Oops, something went wrong, please try again later"})
		return
	}

	c.HTML(200, "messages.html", gin.H{"messages": messages})
}

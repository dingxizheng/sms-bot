package web

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const pageSize = 30

type Page struct {
	URL     string
	Text    string
	Active  bool
	Current int
}

func pagination(c int, m int) []Page {
	var current = c
	var last = m
	var delta = 2
	var left = current - delta
	var right = current + delta + 1
	var pages []Page
	var pagesWithDots []Page
	var l = -1

	for i := 1; i <= last; i++ {
		if i == 1 || i == last || i >= left && i < right {
			pages = append(pages, Page{Current: i, Text: strconv.Itoa(i)})
		}
	}

	for _, page := range pages {
		i := page.Current
		if l != -1 {
			if i-l == 2 {
				pagesWithDots = append(pagesWithDots, Page{Current: l + 1, Text: strconv.Itoa(l + 1)})
			} else if i-l != 1 {
				pagesWithDots = append(pagesWithDots, Page{Current: i - 1, Text: "..."})
			}
		}
		pagesWithDots = append(pagesWithDots, page)
		l = i
	}

	return pagesWithDots
}

func MountIndexController(router *gin.Engine) {
	router.GET("/", FindNumbers)
	router.GET("/home", FindNumbers)
	router.GET("/phone-numbers", FindNumbers)
	router.GET("/phone-numbers/:page", FindNumbers)
	router.GET("/index", FindNumbers)

	router.GET("/free-sms-messages/:provider/:providerId", FindMessages)
	router.GET("/messages", FindMessages)
}

func FindNumbers(c *gin.Context) {
	page := c.Param("page")
	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum <= 0 {
		pageNum = 1
	}

	coll := db.Collection("numbers")
	totalNums, _ := coll.CountDocuments(db.DefaultCtx(), bson.M{"status": "online"})
	totalPages := int(math.Ceil(float64(totalNums) / float64(pageSize)))
	cursor, err := coll.Find(db.DefaultCtx(), bson.M{"status": "online"}, options.Find().SetLimit(pageSize).SetSkip(int64((pageNum-1)*pageSize)))

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

	pages := pagination(pageNum, totalPages)
	for idx, _ := range pages {
		pages[idx].URL = fmt.Sprintf("/phone-numbers/%d", pages[idx].Current)
		if pageNum == pages[idx].Current {
			pages[idx].Active = true
		} else {
			pages[idx].Active = false
		}
	}
	previousURL := fmt.Sprintf("/phone-numbers/%d", pageNum-1)
	nextURL := fmt.Sprintf("/phone-numbers/%d", pageNum+1)

	c.HTML(200, "index.html", gin.H{"numbers": numbers, "hasPrevious": pageNum > 1, "hasNext": pageNum < totalPages, "previousURL": previousURL, "nextURL": nextURL, "pages": pages})
}

// FindMessages - returns the most recent messages of the given number
func FindMessages(c *gin.Context) {
	provider := c.Param("provider")
	providerID := c.Param("providerId")
	coll := db.Collection("numbers")
	var phoneNumber models.PhoneNumber
	err := coll.FindOne(db.DefaultCtx(), bson.M{"provider": provider, "provider_id": providerID}).Decode(&phoneNumber)

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
		coll.UpdateOne(db.DefaultCtx(), bson.M{"_id": phoneNumber.ID}, bson.M{"$set": bson.M{"next_read_at": nextRunAt}})
	}

	if err != nil {
		log.Printf("Failed to decode phone numbers, error: %+v", err)
		c.HTML(500, "messages.html", gin.H{"error": "Oops, something went wrong, please try again later"})
		return
	}

	c.HTML(200, "messages.html", gin.H{"messages": messages, "number": phoneNumber})
}

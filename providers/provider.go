package providers

import (
	"log"
	"strings"
	"time"

	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/dingxizheng/sms-bot/providers/receivesmss"
	"github.com/dingxizheng/sms-bot/providers/sms24"
	"github.com/dingxizheng/sms-bot/providers/yinsiduanxin"
	"github.com/dingxizheng/sms-bot/web"
	"go.mongodb.org/mongo-driver/bson"
)

// ProviderClient - returns provier client for given phone number
func ProviderClient(num models.PhoneNumber) models.SMSProvider {
	if num.Provider == yinsiduanxin.ProviderName {
		return &yinsiduanxin.Client{}
	} else if num.Provider == receivesmss.ProviderName {
		return &receivesmss.Client{}
	} else if num.Provider == sms24.ProviderName {
		return &sms24.Client{}
	}

	return nil
}

// ReadMessages - read phone messages
func ReadMessages(num models.PhoneNumber) {
	log.Printf("Reading messages for number(%v) %v at %v", num.Provider, num.RawNumber, num.NextReadAt)

	var currentJob models.PhoneNumber
	coll := db.Collection("numbers")
	coll.FindOneAndUpdate(db.DefaultCtx(), bson.M{"_id": num.ID, "running": bson.M{"$ne": true}}, bson.M{"$set": bson.M{"running": true}}).Decode(&currentJob)

	if len(currentJob.Number) == 0 {
		log.Printf("Job %v is already running, skip.", num.RawNumber)
		return
	}

	client := ProviderClient(num)
	messages := client.FetchMessages(num.ProviderID, 0)
	log.Printf("%d messages found for number: %v", len(messages), num.RawNumber)

	updates := bson.M{
		"messages":     messages,
		"last_read_at": time.Now(),
		"running":      false,
	}

	if len(messages) == 0 {
		updates = bson.M{
			"last_read_at": time.Now(),
			"running":      false,
		}
	}

	unsets := bson.M{
		"next_read_at": "",
	}

	coll.UpdateOne(
		db.DefaultCtx(),
		bson.M{"_id": num.ID},
		bson.D{
			{"$set", updates},
			{"$unset", unsets},
		},
	)

	if len(messages) == 0 {
		return
	}

	go func() {
		providerNumber := num.Provider + "|" + num.ProviderID
		for _, client := range web.NumberChannels {
			if strings.EqualFold(client.Number, providerNumber) {
				client.Channel <- 1
			}
		}
	}()
}

// ReadMessagesForNewNumbers - reads recent messages for newly added phone numbers
func ReadMessagesForNewNumbers() {
	coll := db.Collection("numbers")
	for {
		cursor, _ := coll.Find(db.DefaultCtx(), bson.M{"status": "online", "last_read_at": nil, "provider": "ReceiveSmss"})
		for cursor.Next(db.DefaultCtx()) {
			num := models.PhoneNumber{}
			cursor.Decode(&num)
			time.Sleep(2 * time.Second)
			go ReadMessages(num)
		}
	}
}

// ReadMessagesForScheduledNumbers - reads most recent messages for a given number
func ReadMessagesForScheduledNumbers() {
	coll := db.Collection("numbers")
	for {
		time.Sleep(2 * time.Second)
		cursor, _ := coll.Find(db.DefaultCtx(), bson.M{"status": "online", "next_read_at": bson.M{"$lte": time.Now()}})
		for cursor.Next(db.DefaultCtx()) {
			num := models.PhoneNumber{}
			cursor.Decode(&num)
			go ReadMessages(num)
		}
	}
}

// ScanPhoneNumbers - finds nubmers
func ScanPhoneNumbers() {
	client1 := receivesmss.Client{}
	client2 := sms24.Client{}
	for {
		time.Sleep(1 * time.Hour)
		client2.StartCrawling()
		client1.StartCrawling()
	}
}

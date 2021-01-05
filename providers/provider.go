package providers

import (
	"log"
	"time"

	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/dingxizheng/sms-bot/providers/receivesmss"
	"github.com/dingxizheng/sms-bot/providers/yinsiduanxin"
	"go.mongodb.org/mongo-driver/bson"
)

func ProviderClient(num models.PhoneNumber) models.SMSProvider {
	if num.Provider == yinsiduanxin.ProviderName {
		return &yinsiduanxin.Client{}
	} else if num.Provider == receivesmss.ProviderName {
		return &receivesmss.Client{}
	}

	return nil
}

func ReadMessages(num models.PhoneNumber) {
	log.Printf("Reading messages for number: %v, provider: %v", num.RawNumber, num.Provider)

	var currentJob models.PhoneNumber
	coll := db.Collection("numbers")
	coll.FindOneAndUpdate(db.DefaultCtx(), bson.M{"_id": num.ID, "running": bson.M{"$ne": true}}, bson.M{"$set": bson.M{"running": true}}).Decode(&currentJob)

	defer func() {
		coll.UpdateOne(db.DefaultCtx(), bson.M{"_id": num.ID}, bson.D{
			{"$set", bson.M{"running": false}},
		})
	}()

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
			"next_read_at": time.Now().Add(5 * time.Second),
			"running":      false,
		}
	}

	unsets := bson.M{
		"next_read_at": "",
	}

	coll.UpdateOne(db.DefaultCtx(), bson.M{"_id": num.ID}, bson.D{
		{"$set", updates},
		{"$unset", unsets},
	})
}

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

func ReadMessagesForScheduledJobs() {
	coll := db.Collection("numbers")
	for {
		time.Sleep(1 * time.Second)
		cursor, _ := coll.Find(db.DefaultCtx(), bson.M{"status": "online", "next_read_at": bson.M{"$gte": time.Now()}})
		for cursor.Next(db.DefaultCtx()) {
			num := models.PhoneNumber{}
			cursor.Decode(&num)
			go ReadMessages(num)
		}
	}
}

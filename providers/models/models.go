package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PhoneNumber struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Provider    string             `json:"provider" bson:"provider"`
	RawNumber   string             `json:"raw_number" bson:"raw_number"`
	ProviderID  string             `json:"provider_id" bson:"provider_id"`
	Number      string             `json:"number" bson:"number"`
	Country     string             `json:"country" bson:"country"`
	CountryCode int                `json:"country_code" bson:"country_code"`
	Status      string             `json:"status" bson:"status"`
	URL         string             `json:"url" bson:"url"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at,omitempty"`
	LastReadAt  time.Time          `json:"last_read_at" bson:"last_read_at,omitempty"`
	NextReadAt  time.Time          `json:"next_read_at" bson:"next_read_at,omitempty"`
	Messages    []Message          `json:"messages" bson:"messages,omitempty"`
	CountryName string             `json:"-" bson:"-"`
}

type Message struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	Provider      string             `json:"provider" bson:"provider"`
	PhoneNumberID string             `json:"number_id" bson:"number_id"`
	From          string             `json:"from" bson:"from"`
	Text          string             `json:"text" bson:"text"`
	ReceivedAt    time.Time          `json:"received_at" bson:"received_at"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
}

type SMSProvider interface {
	Name() string
	StartCrawling()
	FetchNumbers(page int) []PhoneNumber
	FetchMessages(number string, page int) []Message
}

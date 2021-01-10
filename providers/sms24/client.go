package sms24

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ttacon/libphonenumber"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/time/rate"

	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/httpclient"
	"github.com/dingxizheng/sms-bot/providers/models"
)

const (
	smsNumberListApi = "https://sms24.me/numbers-%d"
	smsMessageApi    = "https://sms24.me/number-%s-%d"
	timeLayout       = "2006-01-02 15:04:05-0700"
	maxPage          = 30
)

type Client struct{}

var rl = rate.NewLimiter(rate.Every(1*time.Second), 1)
var httpClient = httpclient.NewClient(rl)

// Name of the provider
const ProviderName = "sms"

func setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_1_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	req.Header.Set("Referer", "https://www.google.com/")
}

// Name returns the name of current provider
func (pv *Client) Name() string {
	return ProviderName
}

func (pv *Client) PhoneNubmerURL(number models.PhoneNumber) string {
	return "https://sms24.me/number-" + number.ProviderID + "-%d"
}

func (pv *Client) StartCrawling() {
	coll := db.Collection("numbers")
	log.Printf("Start crawling numbers from site: SMS24...")
	for page := 1; page <= maxPage; page++ {
		numbers := pv.FetchNumbers(fmt.Sprintf(smsNumberListApi, page), page)
		for _, number := range numbers {
			var existingNum models.PhoneNumber
			filter := bson.M{"provider": number.Provider, "provider_id": number.ProviderID}
			log.Printf("Looking for existing number %+v ...", filter)
			err := coll.FindOne(db.DefaultCtx(), filter).Decode(&existingNum)
			if err != nil && err != mongo.ErrNoDocuments {
				log.Println("Failed to decode document.")
				continue
			}

			if err == mongo.ErrNoDocuments {
				log.Printf("Inserting number %+v", number)
				number.ID = primitive.NewObjectID()
				number.CreatedAt = time.Now()
				coll.InsertOne(db.DefaultCtx(), number)
			} else {
				log.Printf("Updating number %+v", number)
				coll.UpdateOne(db.DefaultCtx(), filter, bson.M{"status": number.Status})
			}
		}

		time.Sleep(20 * time.Second)
	}
}

func (pv *Client) FetchNumbers(url string, page int) []models.PhoneNumber {
	numbers := make([]models.PhoneNumber, 0)
	// Load the HTML document
	doc, err := models.FetchPage(httpClient, url, setDefaultHeaders)
	if err != nil {
		return numbers
	}

	doc.Find("div.country").Each(func(i int, s *goquery.Selection) {
		status := "online"
		number := s.Find("a span").Text()
		numberURL := s.Find("a").AttrOr("href", "nil")
		id := strings.Replace(numberURL, "/number-", "", 1)

		if len(number) == 0 {
			return
		}

		num, err := libphonenumber.Parse(number, "")
		if err != nil {
			log.Printf("Failed to parse number: %s, error: %+v", number, err)
		}

		regionNumber := libphonenumber.GetRegionCodeForNumber(num)
		countryCode := libphonenumber.GetCountryCodeForRegion(regionNumber)
		nationalNum := libphonenumber.GetNationalSignificantNumber(num)

		numbers = append(numbers, models.PhoneNumber{
			Provider:    pv.Name(),
			ProviderID:  id,
			RawNumber:   number,
			Number:      nationalNum,
			Country:     regionNumber,
			CountryCode: countryCode,
			Status:      status,
		})
	})

	return numbers
}

// FetchMessages returns list of SMS messages
func (pv *Client) FetchMessages(url string, page int) []models.Message {
	messages := make([]models.Message, 0)

	for step := 1; step < 2; step++ {
		url := fmt.Sprintf(url, step)
		// Load the HTML document
		doc, err := models.FetchPage(httpClient, url, setDefaultHeaders)
		if err != nil {
			return messages
		}

		doc.Find("div[data-created].text-muted.text-center + div").Each(func(i int, s *goquery.Selection) {
			receivedAt := s.PrevFiltered("div[data-created]").AttrOr("data-created", "0")
			from := s.Find("div a").Text()
			rawText := s.Text()
			if len(strings.TrimSpace(from)) == 0 {
				return
			}
			text := strings.Replace(rawText, "From: "+from, "", 1)

			receivedAtTimestamp, err := strconv.ParseInt(receivedAt, 10, 64)
			if err != nil {
				panic(err)
			}
			receivedAtTime := time.Unix(receivedAtTimestamp, 0)

			messages = append(messages, models.Message{
				Provider:   pv.Name(),
				From:       from,
				Text:       strings.TrimSpace(text),
				ReceivedAt: receivedAtTime,
			})
		})
	}

	return messages
}

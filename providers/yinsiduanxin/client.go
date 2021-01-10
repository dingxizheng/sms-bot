package yinsiduanxin

import (
	"fmt"
	"log"
	"net/http"
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
	smsNumberListApi = "https://www.yinsiduanxin.com/phone-number/page/%d.html"
	smsMessageApi    = "https://www.yinsiduanxin.com/china-phone-number/verification-code-%s/0.html"
	timeLayout       = "2006-01-02 15:04:05-0700"
	maxPage          = 100
)

type Client struct{}

var rl = rate.NewLimiter(rate.Every(1*time.Second), 2)
var httpClient = httpclient.NewClient(rl)

const ProviderName = "ysdx"

func setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_1_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	req.Header.Set("Referer", "https://www.google.com/")
}

// Name returns the name of current provider
func (pv *Client) Name() string {
	return ProviderName
}

func (pv *Client) PhoneNubmerURL(number models.PhoneNumber) string {
	return fmt.Sprintf(smsMessageApi, number.ProviderID)
}

func (pv *Client) StartCrawling() {
	coll := db.Collection("numbers")
	log.Printf("Start crawling numbers from site: YinSiDuanXin...")
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

	doc.Find("div.layui-card").Each(func(i int, s *goquery.Selection) {
		status := "online"
		rawStatus := s.Find("div.layui-card-header span:nth-child(2)").Text()
		number := s.Find("div.layui-card-body p.card-phone a").Text()
		id := s.Find("div.layui-card-body p.card-phone").AttrOr("id", "nil")

		if len(number) == 0 {
			return
		}

		if strings.EqualFold(rawStatus, "离线") {
			status = "offline"
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
	// Load the HTML document
	doc, err := models.FetchPage(httpClient, url, setDefaultHeaders)
	if err != nil {
		return messages
	}

	doc.Find("table.layui-table tbody tr").Each(func(i int, s *goquery.Selection) {
		from := s.Find("td:nth-child(1) a").Text()
		text := s.Find("td:nth-child(2)").Text()
		receivedAt := s.Find("td:nth-child(3)").Text()

		if len(strings.TrimSpace(from)) == 0 {
			return
		}

		receivedAtTime, _ := time.Parse(timeLayout, receivedAt+"+0800")
		messages = append(messages, models.Message{
			Provider:   pv.Name(),
			From:       from,
			Text:       strings.TrimSpace(text),
			ReceivedAt: receivedAtTime,
		})
	})

	return messages
}

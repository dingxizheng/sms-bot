package yunjiema

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gosimple/slug"
	"github.com/karrick/tparse/v2"
	"github.com/ttacon/libphonenumber"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/time/rate"

	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/httpclient"
	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/dingxizheng/sms-bot/utils"
)

const (
	smsNumberListAPI = "https://yunjiema.net/%s/%d.html"
	smsMessageAPI    = "https://sms24.me/meiguohaoma-%s-%d"
	timeLayout       = "2006-01-02 15:04:05-0700"
	maxPage          = 3
)

var timeUnitMap = map[string]string{
	"mins": "min",
	"secs": "sec",
}

type Client struct{}

var rl = rate.NewLimiter(rate.Every(1*time.Second), 1)
var httpClient = httpclient.NewClient(rl)

// ProviderName - name of the provider
const ProviderName = "yjm"

func setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_1_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	req.Header.Set("Referer", "https://www.google.com/")
}

// Name returns the name of current provider
func (pv *Client) Name() string {
	return ProviderName
}

func (pv *Client) PhoneNubmerURL(number models.PhoneNumber) string {
	return number.URL
}

func (pv *Client) StartCrawling() {
	coll := db.Collection("numbers")
	log.Printf("Start crawling numbers from site: YunJieMa...")
	for page := 1; page <= maxPage; page++ {
		countryURLs := pv.FetchCountryURLs(fmt.Sprintf("https://yunjiema.net/guojiadiqu/%d.html", page), 0)

		for _, coutryURL := range countryURLs {
		fetchNumbersLoop:
			for numberPage := 1; numberPage <= 1000; numberPage++ {
				numbers := pv.FetchNumbers(coutryURL+fmt.Sprintf("%d.html", numberPage), page)
				if len(numbers) == 0 {
					break fetchNumbersLoop
				}

				for _, number := range numbers {
					var existingNum models.PhoneNumber
					filter := bson.M{"provider": number.Provider, "provider_id": number.ProviderID}
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
			}
		}
		time.Sleep(20 * time.Second)
	}
}

func (pv *Client) FetchCountryURLs(url string, page int) []string {
	countryURLs := make([]string, 0)
	// Load the HTML document
	doc, err := models.FetchPage(httpClient, url, setDefaultHeaders)

	if err != nil {
		return countryURLs
	}

	doc.Find("div.number-boxes-item").Each(func(i int, s *goquery.Selection) {
		countryURL := s.Find("div.row:nth-child(3) div a").AttrOr("href", "nil")

		if len(countryURL) == 0 {
			return
		}
		countryURLs = append(countryURLs, countryURL)
	})

	return countryURLs
}

func (pv *Client) FetchNumbers(url string, page int) []models.PhoneNumber {
	numbers := make([]models.PhoneNumber, 0)
	// Load the HTML document
	doc, err := models.FetchPage(httpClient, url, setDefaultHeaders)
	if err != nil {
		return numbers
	}

	doc.Find("div.number-boxes-item").Each(func(i int, s *goquery.Selection) {
		status := "online"
		number := s.Find(".number-boxes-item-number").Text()
		numberURL := s.Find("a.number-boxes-item-button").AttrOr("href", "nil")
		id := strings.Split(numberURL, "/")[4]

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
			URL:         numberURL,
			RawNumber:   number,
			Number:      nationalNum,
			Country:     regionNumber,
			CountryCode: countryCode,
			CountryName: utils.FindCountryName(regionNumber),
			CountrySlug: slug.Make(utils.FindCountryName(regionNumber)),
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

	doc.Find("div.row.border-bottom.table-hover").Each(func(i int, s *goquery.Selection) {
		receivedAt := s.Find("div.mobile_hide:nth-child(2)").Text()
		receivedAtTime := time.Now()
		from := s.Find("div:nth-child(1) div.mobile_hide").Text()
		text := s.Find("div:nth-child(3)").Text()

		if len(strings.TrimSpace(from)) == 0 || strings.EqualFold(from, "ADS") {
			return
		}

		if len(receivedAt) != 0 {
			parts := strings.Split(receivedAt, " ")
			durationUnit, ok := timeUnitMap[parts[1]]
			if !ok {
				durationUnit = parts[1]
			}
			receivedAtTime, err = tparse.AddDuration(time.Now(), fmt.Sprintf("-%s%s", parts[0], durationUnit))
		}

		messages = append(messages, models.Message{
			Provider:   pv.Name(),
			From:       strings.TrimSpace(from),
			Text:       strings.Join(strings.Fields(strings.TrimSpace(text)), " "),
			ReceivedAt: receivedAtTime,
		})
	})

	return messages
}

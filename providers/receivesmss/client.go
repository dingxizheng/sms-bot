package receivesmss

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dingxizheng/sms-bot/db"
	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/karrick/tparse/v2"
	"github.com/ttacon/libphonenumber"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	smsNumberListApi = "https://receive-smss.com/"
	smsMessageApi    = "https://receive-smss.com/sms/%s/"
	timeLayout       = "2006-01-02 15:04:05-0700"
	maxPage          = 100
)

type Client struct{}

var httpClient = &http.Client{}

const ProviderName = "ReceiveSmss"

func setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_1_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	req.Header.Set("Referer", "https://receive-smss.com/sms/")
	req.Header.Set("Cookie", "__cfduid=d2fc1f5d650d5249a10a45e1d0bf572381609708106; PHPSESSID=e4ngj48e2mdtf6rt2ftcal75pd;")
}

// Name returns the name of current provider
func (pv *Client) Name() string {
	return ProviderName
}

func (pv *Client) StartCrawling() {
	coll := db.Collection("numbers")
	log.Printf("Start fetching numbers...")
	numbers := pv.FetchNumbers(0)
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
			log.Printf("Replacing number %+v", number)
			coll.UpdateOne(db.DefaultCtx(), filter, bson.M{"status": number.Status})
		}
	}
}

// FetchNumbers returns phoner numbers from the page
func (pv *Client) FetchNumbers(page int) []models.PhoneNumber {
	url := smsNumberListApi
	log.Printf("Fetching numbers from URL %v", url)

	req, err := http.NewRequest("GET", url, nil)
	setDefaultHeaders(req)

	res, _ := httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to load page, error: %v", err.Error())
		return []models.PhoneNumber{}
	}
	defer res.Body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("Failed to parse document, error: %v", err.Error())
	}

	numbers := make([]models.PhoneNumber, 0)

	doc.Find("div.number-boxes div.number-boxes-item").Each(func(i int, s *goquery.Selection) {
		status := "online"
		numberURL := s.Find("a").AttrOr("href", "nil")
		number := s.Find(".number-boxes-itemm-number").Text()
		id := strings.Replace(numberURL, "/sms/", "", 1)

		if len(number) == 0 || strings.Contains(numberURL, "register") {
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

// FetchMessages returns the messages of the page
func (pv *Client) FetchMessages(number string, page int) []models.Message {
	pageURL := fmt.Sprintf(smsMessageApi, number)
	log.Printf("Fetching messages from url %v", pageURL)
	req, err := http.NewRequest("GET", pageURL, nil)
	setDefaultHeaders(req)
	res, _ := httpClient.Do(req)
	if err != nil {
		log.Print(err)
	}
	defer res.Body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Print(err)
	}

	messages := make([]models.Message, 0)

	doc.Find(".main-content table tr").Each(func(i int, s *goquery.Selection) {
		from := s.Find("td:nth-child(2)").Text()
		text := s.Find("td:nth-child(4)").Text()
		receivedAt := s.Find("td:nth-child(5)").Text()
		receivedAtTime := time.Now()

		if len(strings.TrimSpace(from)) == 0 {
			return
		}

		if len(receivedAt) != 0 {
			parts := strings.Split(receivedAt, " ")
			receivedAtTime, err = tparse.AddDuration(time.Now(), fmt.Sprintf("-%s%s", parts[0], parts[1]))
		}

		messages = append(messages, models.Message{
			Provider:      pv.Name(),
			PhoneNumberID: number,
			From:          from,
			Text:          strings.TrimSpace(text),
			ReceivedAt:    receivedAtTime,
		})
	})

	return messages
}

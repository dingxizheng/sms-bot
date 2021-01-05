package main

import ( // "github.com/go-resty/resty/v2"
	// Load variables from .env file

	"github.com/dingxizheng/sms-bot/providers"
	"github.com/dingxizheng/sms-bot/web"
	_ "github.com/joho/godotenv/autoload"
)

func main() {

	// num, err := libphonenumber.Parse("+86 17199741419", "")
	// regionNumber := libphonenumber.GetRegionCodeForNumber(num)
	// // countryCode := libphonenumber.GetCountryCodeForRegion(regionNumber)
	// log.Printf("GOOD nubmer: %v", regionNumber)

	// client2 := receivesmss.Client{}
	// client2.StartCrawling()
	// messages := client2.FetchMessages("18632801187", 0)
	// log.Printf("Messages: %+v", messages)
	// client := yinsiduanxin.Client{}

	// // client.StartCrawling()
	// // log.Printf("Numbers: %+v", numbers)

	// messages := client.FetchMessages("7044570075", 0)
	// log.Printf("Messages: %+v", messages)

	// go providers.ReadMessagesForNewNumbers()
	go providers.ReadMessagesForScheduledNumbers()
	web.Start()
}

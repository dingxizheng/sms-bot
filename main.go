package main

import (
	"github.com/dingxizheng/sms-bot/providers"
	"github.com/dingxizheng/sms-bot/web"

	// Load variables from .env file
	_ "github.com/joho/godotenv/autoload"
)

func main() {

	// num, err := libphonenumber.Parse("+86 17199741419", "")
	// regionNumber := libphonenumber.GetRegionCodeForNumber(num)
	// // countryCode := libphonenumber.GetCountryCodeForRegion(regionNumber)
	// log.Printf("GOOD nubmer: %v", regionNumber)

	// client2 := yunjiema.Client{}
	// // client2.StartCrawling()
	// messages := client2.FetchCountryURLs("https://yunjiema.net/guojiadiqu/1.html", 0)
	// log.Printf("Messages: %+v", messages)
	// client := yinsiduanxin.Client{}

	// client2.StartCrawling()

	// // client.StartCrawling()
	// // log.Printf("Numbers: %+v", numbers)

	// messages := client.FetchMessages("7044570075", 0)
	// log.Printf("Messages: %+v", messages)

	// go providers.ReadMessagesForNewNumbers()
	go providers.ScanPhoneNumbers()
	go providers.ReadMessagesForScheduledNumbers()
	web.Start()
}

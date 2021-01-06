package models

import (
	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/dingxizheng/sms-bot/httpclient"
)

// FetchPage - returns parsed html dom
func FetchPage(httpClient *httpclient.RLHTTPClient, url string, setDefaultHeaders func(*http.Request)) (*goquery.Document, error) {
	log.Printf("Fetching page from URL %v", url)

	req, err := http.NewRequest("GET", url, nil)
	setDefaultHeaders(req)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Load the HTML document
	return goquery.NewDocumentFromReader(res.Body)
}

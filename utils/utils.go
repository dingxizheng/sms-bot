package utils

import (
	"strconv"

	"github.com/pariz/gountries"
)

var query = gountries.New()
var countryCache = map[string]string{}

// Page - page object
type Page struct {
	URL     string
	Text    string
	Active  bool
	Current int
}

// Pagination - paginate
func Pagination(c int, m int) []Page {
	var current = c
	var last = m
	var delta = 2
	var left = current - delta
	var right = current + delta + 1
	var pages []Page
	var pagesWithDots []Page
	var l = -1

	for i := 1; i <= last; i++ {
		if i == 1 || i == last || i >= left && i < right {
			pages = append(pages, Page{Current: i, Text: strconv.Itoa(i)})
		}
	}

	for _, page := range pages {
		i := page.Current
		if l != -1 {
			if i-l == 2 {
				pagesWithDots = append(pagesWithDots, Page{Current: l + 1, Text: strconv.Itoa(l + 1)})
			} else if i-l != 1 {
				pagesWithDots = append(pagesWithDots, Page{Current: i - 1, Text: "..."})
			}
		}
		pagesWithDots = append(pagesWithDots, page)
		l = i
	}

	return pagesWithDots
}

// FindCountryName - gets coutry common name by code
func FindCountryName(countryCode string) string {
	name := countryCache[countryCode]
	if len(name) != 0 {
		return name
	}
	country, _ := query.FindCountryByAlpha(countryCode)
	countryCache[countryCode] = country.Name.Common
	return country.Name.Common
}

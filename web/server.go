package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dingxizheng/sms-bot/providers/models"
	"github.com/dingxizheng/sms-bot/utils"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gosimple/slug"
	"github.com/thedevsaddam/renderer"
	"github.com/xeonx/timeago"
)

var rnd *renderer.Render

type H map[string]interface{}

func formatAsTimeAgo(t time.Time) string {
	return timeago.English.Format(t)
}

func timeString(t time.Time) string {
	return t.Format("2006-01-02T15:04:05-07:00")
}

func phumberNumberURL(number models.PhoneNumber) string {
	return fmt.Sprintf("/phone-number-%v-%v", number.Provider, number.ProviderID)
}

// Start - starts the web server
func Start() {

	rnd = renderer.New(
		renderer.Options{
			ParseGlobPattern: "templates/*.html",
			FuncMap: []template.FuncMap{
				{
					"formatAsTimeAgo":  formatAsTimeAgo,
					"timeString":       timeString,
					"phumberNumberURL": phumberNumberURL,
				},
			},
		},
	)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	router.Use(middleware.Timeout(20 * time.Second))

	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/favicon.ico")
	})

	router.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/robots.txt")
	})

	router.Get("/privacy", func(w http.ResponseWriter, r *http.Request) {
		rnd.HTML(w, http.StatusOK, "privacy.html", H{})
	})

	router.Get("/.sitemap.xml", GenerateSiteMapFile)

	router.Get("/ws", WSHandler)

	// Mount phone number controllers
	router.Get("/", renderNumbers)

	// router.GET("/home", FindNumbers)
	router.Get("/countries", FindCountries)

	// TODO remove in next release
	router.Get("/phone-numbers/{country}/{page}", func(w http.ResponseWriter, r *http.Request) {
		page := chi.URLParam(r, "page")
		country := chi.URLParam(r, "country")
		if len(country) == 0 {
			country = "all"
		} else {
			country = slug.Make(utils.FindCountryName(country))
		}
		FindNumbers(w, r, page, country)
	})
	// TODO remove in next release
	router.Get("/free-sms-messages/{provider}/{providerId}", FindMessages)

	router.Get("/phone-numbers/{country}", renderNumbers)

	router.Get("/phone-numbers", renderNumbers)

	router.Get("/phone-number-{provider}-{providerId}", FindMessages)

	// router.GET("/phone-numbers/:country/:page", FindNumbers)

	// 404 Not Found
	// Create a route along /files that will serve contents from
	// the ./data/ folder.

	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		rnd.HTML(w, http.StatusNotFound, "404.html", H{})
	})

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	http.ListenAndServe(":"+port, router)
}

func renderNumbers(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	country := chi.URLParam(r, "country")
	if len(country) == 0 {
		country = "all"
	}
	FindNumbers(w, r, page, country)
}

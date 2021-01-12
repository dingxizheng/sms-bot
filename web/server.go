package web

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thinkerou/favicon"
	"github.com/xeonx/timeago"
)

func formatAsTimeAgo(t time.Time) string {
	return timeago.English.Format(t)
}

func timeString(t time.Time) string {
	return t.Format("2006-01-02T15:04:05-07:00")
}

// Start - starts the web server
func Start() {
	router := gin.Default()

	// Template helper functions
	router.SetFuncMap(map[string]interface{}{
		"formatAsTimeAgo": formatAsTimeAgo,
		"timeString":      timeString,
	})

	// Load templates
	router.LoadHTMLGlob("templates/*")

	// Robots.txt
	router.StaticFile("/robots.txt", "./public/robots.txt")
	// Handle favicon requset
	router.Use(favicon.New("./favicon.ico"))
	router.Use(gin.Recovery())

	// Handle not found
	router.NoRoute(func(c *gin.Context) {
		c.HTML(404, "404.html", gin.H{})
	})

	// Mount ws handler
	MountWSController(router)

	// Mount controllers
	MountIndexController(router)

	// Mount phone number controllers
	MountPhoneNumberController(router)

	router.Run()
}

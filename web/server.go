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

// Start - starts the web server
func Start() {
	router := gin.Default()
	router.SetFuncMap(map[string]interface{}{
		"formatAsTimeAgo": formatAsTimeAgo,
	})
	router.LoadHTMLGlob("templates/*")
	// Mount favicon
	router.Use(favicon.New("./favicon.ico"))

	// Not found
	router.NoRoute(func(c *gin.Context) {
		c.HTML(404, "404.html", gin.H{})
	})

	// Mount controllers
	MountIndexController(router)

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	router.Run()
}

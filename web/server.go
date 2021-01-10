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

	// Template helper functions
	router.SetFuncMap(map[string]interface{}{
		"formatAsTimeAgo": formatAsTimeAgo,
	})

	// Load templates
	router.LoadHTMLGlob("templates/*")

	// Handle favicon requset
	router.Use(favicon.New("./favicon.ico"))

	// Handle not found
	router.NoRoute(func(c *gin.Context) {
		c.HTML(404, "404.html", gin.H{})
	})

	// Mount ws handler
	MountWSController(router)

	// Mount controllers
	MountIndexController(router)

	router.Run()
}

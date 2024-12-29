package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
  "os"
	"time"

	"github.com/gin-gonic/gin"
)

type Tip struct {
	ID  int    `json:"id"`
	Tip string `json:"tip"`
}

func main() {
	tips, err := loadTips("./web/tips.json")
	if err != nil {
		panic(err)
	}

  r := gin.Default()

  r.Static("/static", "./static")

  r.GET("/", func(c *gin.Context) {
    c.File("static/index.html")
  })

	// Route to get the daily tip
	r.GET("/daily-tip", func(c *gin.Context) {
		today := time.Now().Day()
		dailyTip := tips[today%len(tips)]
		c.JSON(http.StatusOK, dailyTip)
	})

	// Route to get a random tip
	r.GET("/random-tip", func(c *gin.Context) {
		rand.Seed(time.Now().UnixNano())
		randomTip := tips[rand.Intn(len(tips))]
		c.JSON(http.StatusOK, randomTip)
	})


	// Start server
	r.Run(":8080")
}

func loadTips(filename string) ([]Tip, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read tips file: %v", err)
	}

	var tips []Tip
	if err := json.Unmarshal(file, &tips); err != nil {
		return nil, fmt.Errorf("failed to parse tips: %v", err)
	}
	return tips, nil
}


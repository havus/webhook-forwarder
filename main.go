package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	port := os.Getenv("PORT")

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.POST("/google-chat", postGoogleChat)

	r.Run(fmt.Sprintf(":%s", port))
}

// postAlbums adds an album from JSON received in the request body.
func postGoogleChat(c *gin.Context) {
	var requestBody map[string]interface{}

	// METHODE 2:
	// requestBodyBytes, _ := ioutil.ReadAll(c.Request.Body)
	// if err := json.Unmarshal(requestBodyBytes, &requestBody); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	// 	return
	// }

	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	requestHeader := c.Request.Header
	requestQueryParams := c.Request.URL.Query()
	scriptId := requestQueryParams.Get("script_id")

	requestQueryParams.Del("script_id")
	requestBody["forwarder_header"] = requestHeader

	// TODO: add validation
	// if _, ok := requestHeader["space_id"]; !ok {
	// 	c.JSON(http.StatusOK, gin.H{"message": "'space_id' query parameter is required."})
	// 	return
	// }

	ok, err := sendToGoogleChat(scriptId, &requestQueryParams, &requestBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if !ok {
		c.JSON(http.StatusOK, gin.H{"message": "not ok"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func sendToGoogleChat(scriptId string, requestQueryStr *url.Values, requestBody *map[string]interface{}) (bool, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	fullUrl := fmt.Sprintf("https://script.google.com/macros/s/%s/exec?%s&", scriptId, requestQueryStr.Encode())

	marshalled, err := json.Marshal(requestBody)
	if err != nil {
		return false, err
	}

	request, err := http.NewRequest("POST", fullUrl, bytes.NewReader(marshalled))
	if err != nil {
		return false, err
	}

	response, err := client.Do(request)
	if err != nil {
		return false, err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		responseBodyBytes, _ := io.ReadAll(response.Body)
		sendErrorNotif(fmt.Sprintf("status %d: %s", response.StatusCode, string(responseBodyBytes)))
	}

	return true, nil
}

func sendErrorNotif(message string) {
	var client = &http.Client{Timeout: 60 * time.Second}

	webhookErrorNotifUrl := os.Getenv("ERROR_NOTIF_URL")

	marshalled, err := json.Marshal(gin.H{"text": fmt.Sprintf("FROM WEBHOOK\n%s", message)})
	if err != nil {
		// TODO:
		return
	}

	request, err := http.NewRequest("POST", webhookErrorNotifUrl, bytes.NewReader(marshalled))
	if err != nil {
		// TODO:
		return
	}

	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
}

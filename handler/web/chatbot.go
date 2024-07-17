package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ChatbotWeb struct {
	embed embed.FS
}

func NewChatbotWeb(embed embed.FS) ChatbotWeb {
	return ChatbotWeb{embed}
}

type Message struct {
	Text string `json:"text"`
}

func getChatbotResponse(input string) (Message, error) {
	apiKey := "AIzaSyAzR5tWcNV_ukoMjFXyAm0yz6EOuZRKUHw"
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/gemini-1.5-flash:generateContent?key=%s", apiKey)

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": input}},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return Message{}, err
	}

	log.Println("Request Body:", string(jsonBody))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Message{}, err
	}

	log.Println("Response Body:", string(body))

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return Message{}, err
	}

	// Menampilkan struktur lengkap dari respons API
	log.Printf("Parsed Response: %+v\n", response)

	// Parsing the response to get the chatbot reply
	candidates, ok := response["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return Message{}, fmt.Errorf("Invalid response format")
	}

	content, ok := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	if !ok {
		return Message{}, fmt.Errorf("Invalid response format")
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return Message{}, fmt.Errorf("Invalid response format")
	}

	reply, ok := parts[0].(map[string]interface{})["text"].(string)
	if !ok {
		return Message{}, fmt.Errorf("Invalid response format")
	}

	return Message{Text: reply}, nil
}

func (ch *ChatbotWeb) Interact(c *gin.Context) {
	var reqBody struct {
		Message string `json:"message"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	userMessage := reqBody.Message

	// Get the chatbot response
	botResponse, err := getChatbotResponse(userMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get chatbot response: %v", err),
		})
		return
	}

	// Send the chatbot response
	c.JSON(http.StatusOK, gin.H{
		"response": botResponse.Text,
	})
}

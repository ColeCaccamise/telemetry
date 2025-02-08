package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/slack-go/slack"
)

func main() {
	fmt.Println("loading environment variables...")
	godotenv.Load() 

	fmt.Println("initializing echo server...")
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		fmt.Println("received request to /")
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.POST("/webhooks/:slug", processWebhook)

	fmt.Printf("starting server on port %s...\n", os.Getenv("PORT"))
	e.Logger.Fatal(e.Start(":" + os.Getenv("PORT")))
}

type ApiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func processWebhook(c echo.Context) error {
	fmt.Printf("processing webhook for slug: %s\n", c.Param("slug"))

	// read request body
	fmt.Println("reading request body...")
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		fmt.Printf("error reading request body: %v\n", err)
		return c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: "failed to read request body",
			Error:   err.Error(),
		})
	}

	// send slack message	
	fmt.Println("preparing slack message...")
	headers := c.Request().Header
	headerJSON, _ := json.MarshalIndent(headers, "", "  ")

	message := fmt.Sprintf("New webhook received:\n `%s %s`\nHeaders:\n```%s```\nBody:\n```%s```",
		c.Request().Method,
		c.Request().URL.Path,
		string(headerJSON),
		string(bodyBytes),
	)

	fmt.Println("sending slack message...")
	err = sendSlackMessage(SlackMessageOpts{
		Message: message,
		Channel: "telemetry",
	})
	if err != nil {
		fmt.Printf("error sending slack message: %v\n", err)
	}

	// create new request to forward with original headers
	fmt.Println("creating forwarding request...")
	req, err := http.NewRequest("POST", os.Getenv("NGROK_DOMAIN")+"/webhooks/"+c.Param("slug"), bytes.NewBuffer(bodyBytes))
	for key, values := range c.Request().Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if err != nil {
		fmt.Printf("error creating new request: %v\n", err)
		return c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: "failed to create new request",
			Error:   err.Error(),
		})
	}

	// forward request with timeout
	fmt.Println("forwarding request...")
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	_, err = client.Do(req)
	if err != nil {
		fmt.Printf("error forwarding request: %v\n", err)
		return c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: "failed to forward request",
			Error:   err.Error(),
		})
	}

	fmt.Println("webhook processed successfully")
	return c.JSON(http.StatusOK, ApiResponse{
		Success: true,
		Message: "Webhook processed successfully",
	})
}

type SlackMessageOpts struct {
	Message string `json:"message"`
	Channel string `json:"channel"`
}

func sendSlackMessage(opts SlackMessageOpts) error {
	fmt.Println("getting slack token...")
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		return fmt.Errorf("slack token not found")
	}

	fmt.Printf("sending message to slack channel: %s\n", opts.Channel)
	api := slack.New(token)

	_, _, err := api.PostMessage(
		opts.Channel,
		slack.MsgOptionText(opts.Message, false),
	)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %v", err)
	}

	fmt.Println("slack message sent successfully")
	return nil
}
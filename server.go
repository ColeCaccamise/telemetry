package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/slack-go/slack"
)

func main() {
	godotenv.Load() 

	e := echo.New()
	e.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
		return key == os.Getenv("API_KEY"), nil
	}))
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.POST("/webhooks/:slug", processWebhook)

	fmt.Println("Starting server on port", os.Getenv("PORT"))
	fmt.Println("API_KEY", os.Getenv("API_KEY"))
	fmt.Println("NGROK_DOMAIN", os.Getenv("NGROK_DOMAIN"))
	fmt.Println("SLACK_TOKEN", os.Getenv("SLACK_TOKEN"))

	e.Logger.Fatal(e.Start(":" + os.Getenv("PORT")))
}

type ApiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func processWebhook(c echo.Context) error {
	// read request body
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: "failed to read request body",
			Error:   err.Error(),
		})
	}

	// send slack message	
	headers := c.Request().Header
	headerJSON, _ := json.MarshalIndent(headers, "", "  ")

	message := fmt.Sprintf("New webhook received:\n `%s %s`\nHeaders:\n```%s```\nBody:\n```%s```",
		c.Request().Method,
		c.Request().URL.Path,
		string(headerJSON),
		string(bodyBytes),
	)

	sendSlackMessage(SlackMessageOpts{
		Message: message,
		Channel: "telemetry",
	}, c)

	// create new request to forward with original headers
	req, _ := http.NewRequest("POST", os.Getenv("NGROK_DOMAIN")+"/webhooks/"+c.Param("slug"), bytes.NewBuffer(bodyBytes))
	for key, values := range c.Request().Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// forward request
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: "failed to forward request",
			Error:   err.Error(),
		})
	}

	// read response body
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: "failed to read response body",
			Error:   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, ApiResponse{
		Success: true,
		Message: "Webhook processed successfully",
	})
}

type SlackMessageOpts struct {
	Message string `json:"message"`
	Channel string `json:"channel"`
}

func sendSlackMessage(opts SlackMessageOpts, c echo.Context) error {
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		return c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: "slack token not found",
			Error:   "SLACK_TOKEN is not set",
		})
	}

	api := slack.New(token)

	_, _, err := api.PostMessage(
		opts.Channel,
		slack.MsgOptionText(opts.Message, false),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: fmt.Sprintf("failed to send slack message: %v", err),
			Error:   err.Error(),
		})
	}

	return nil
}
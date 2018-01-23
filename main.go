package main

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type Invitation struct {
	Email string
}

type Submission struct {
	Data Payload
}

type Payload map[string]interface{}

type Config struct {
	FormVerificationToken  string
	SlackInviteURL         string
	SlackAccessToken       string
}

var submissions = make(chan Submission, 1000)
var invitations = make(chan Invitation, 1000)
var config Config

func main() {
	port := getEnv("PORT", "8080")
	slackOrgName := mustGetEnv("SLACK_ORG_NAME")
	config.FormVerificationToken = mustGetEnv("FORM_VERIFICATION_TOKEN")
	config.SlackInviteURL = fmt.Sprintf("https://%s.slack.com/api/users.admin.invite", slackOrgName)
	config.SlackAccessToken = mustGetEnv("SLACK_ACCESS_TOKEN")

	go invite(config.SlackInviteURL, config.SlackAccessToken, invitations)

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/", index)
	e.POST("/submit", submit)
	e.Logger.Fatal(e.Start(":" + port))
}

func index(c echo.Context) error {
	return c.String(200, "OK")
}

// Receive Submission from External Sources
func submit(c echo.Context) error {
	if c.QueryParam("token") != config.FormVerificationToken {
		return c.NoContent(http.StatusUnauthorized)
	}
	var payload Payload
	if err := c.Bind(&payload); err != nil {
		return err
	}

	if payload["email"] == nil {
		return c.JSON(http.StatusBadRequest, "Missing email parameter")
	}

	invitations <- Invitation{Email: payload["email"].(string)}

	return c.JSON(http.StatusOK, len(submissions))
}

func invite(uri string, token string, jobs chan Invitation) {
	for job := range jobs {
		values := url.Values{"email": {job.Email}, "token": {token}}
		response, err := http.PostForm(uri, values)

		if err != nil {
			fmt.Println(err)
			jobs <- job
			continue
		}

		defer response.Body.Close()
		body, _ := ioutil.ReadAll(response.Body)
		fmt.Printf("%s\n", body)
	}
}

func mustGetEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	panic(fmt.Sprintf("Environment Variable %s missing", key))
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

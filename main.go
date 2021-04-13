package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v2"
)

type AlertType struct {
	Label   string `yaml:"label"`
	Name    string `yaml:"name"`
	Message string `yaml:"message"`
}

type Config struct {
	Username   string `yaml:"username"`
	WebhookURL string `yaml:"webhookURL"`

	AlertTypes []AlertType `yaml:"alertTypes"`
}

var (
	config Config
)

func init() {
	configPath := ""
	flag.StringVar(&configPath, "config", "config.yaml", "--config config.yaml")
	flag.Parse()

	if configPath == "" {
		log.Fatal("unexpected --config")
	}

	f, err := os.Open(configPath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		f.Close()
		log.Fatal(err.Error())
	}
}
func main() {
	r := gin.Default()

	r.LoadHTMLGlob("templates/*.tmpl")

	r.GET("", index)
	r.POST("/alert", alert)
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err.Error())
	}
}

func index(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index.tmpl", config.AlertTypes)
}

type Webhook struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

func alert(ctx *gin.Context) {
	alertType := ctx.PostForm("alertType")
	msg := ""
	for _, t := range config.AlertTypes {
		if t.Name == alertType {
			msg = t.Message
			break
		}
	}
	if msg != "" {
		log.Println("webhook! ", msg)
		c := resty.New()

		b := Webhook{
			Username: config.Username,
			Text:     msg,
		}
		c.R().SetHeaders(map[string]string{
			"Content-Type": "application/json",
		}).SetBody(b).Post(config.WebhookURL)
	}

	ctx.Redirect(http.StatusFound, "")
}

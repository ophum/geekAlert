package main

import (
	"crypto/rand"
	"flag"
	"html"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"

	"gorm.io/driver/sqlite"
)

type AlertType struct {
	gorm.Model
	Label   string `yaml:"label"`
	Message string `yaml:"message"`
}

type Config struct {
	Username         string `yaml:"username"`
	WebhookURL       string `yaml:"webhookURL"`
	DBFilePath       string `yaml:"dbFilePath"`
	HellShakeYanoURL string `yaml:"hellShakeYanoURL"`
	PuiPuiURL        string `yaml:"puiPuiURL"`
}

var (
	config Config
	db     *gorm.DB
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

	db, err = gorm.Open(sqlite.Open(config.DBFilePath), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&AlertType{})
}

func main() {

	r := gin.Default()

	r.Static("/assets", "./assets")
	r.LoadHTMLGlob("templates/*.tmpl")

	r.GET("", index)
	r.POST("/alert", alert)
	r.GET("/create", create)
	r.POST("/store", store)
	r.POST("/delete", delete)
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err.Error())
	}
}

func index(ctx *gin.Context) {
	alertTypes := []AlertType{}
	db.Find(&alertTypes)

	ctx.HTML(http.StatusOK, "index.tmpl", alertTypes)
}

func create(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "create.tmpl", gin.H{})
}

func store(ctx *gin.Context) {
	label := ctx.PostForm("label")
	msg := ctx.PostForm("msg")

	if len(msg) > 32 {
		msg = msg[:32]
	}

	alertType := AlertType{
		Label:   label,
		Message: html.EscapeString(msg),
	}

	if err := db.Create(&alertType).Error; err != nil {

	}
	ctx.Redirect(http.StatusFound, "")
}

type Attachment struct {
	Fallback   string `json:"fallback"`
	AuthorName string `json:"author_name"`
	AuthorLink string `json:"author_link"`
	AuthorIcon string `json:"author_icon"`
	Title      string `json:"title"`
	Footer     string `json:"footer"`
	FooterIcon string `json:"footer_icon"`
	TS         int    `json:"ts"`
}

type Webhook struct {
	Username    string       `json:"username"`
	IconURL     string       `json:"icon_url"`
	Attachments []Attachment `json:"attachments"`
}

func alert(ctx *gin.Context) {
	alertTypeID := ctx.PostForm("alertTypeID")
	var alertType AlertType
	db.First(&alertType, alertTypeID)

	if alertType.Message != "" {
		log.Println("webhook! ", alertType.Message)
		c := resty.New()

		isIncludeVia := ctx.PostForm("isIncludeVia")
		via := "名無しの新卒"
		if isIncludeVia == "on" {
			addr, err := net.LookupAddr(ctx.ClientIP())
			if err != nil {
				via = " (via " + ctx.ClientIP() + ")"
			} else {
				via = " (via " + ctx.ClientIP() + " -> " + addr[0] + ")"
			}
		}

		msg := "その話題... " + alertType.Message + " かも..."
		icon := config.PuiPuiURL
		if n, err := rand.Int(rand.Reader, big.NewInt(100)); err == nil {
			if n.Int64() == 8 {
				if isIncludeVia == "on" {
					via = "ヘルシェイク矢野 " + via
				} else {
					via = "ヘルシェイク矢野"
				}
				icon = config.HellShakeYanoURL
			}
		}

		b := Webhook{
			Username: via,
			IconURL:  icon,
			Attachments: []Attachment{
				{
					Fallback:   msg,
					Title:      msg,
					Footer:     "<https://github.com/ophum/geekAlert|ophum/geekAlert>",
					FooterIcon: "https://slack-imgs.com/?c=1&o1=wi32.he32.si&url=https%3A%2F%2Fgithub.githubassets.com%2Ffavicon.ico",
					TS:         int(time.Now().Unix()),
				},
			},
		}

		c.R().SetHeaders(map[string]string{
			"Content-Type": "application/json",
		}).SetBody(b).Post(config.WebhookURL)
	}

	ctx.Redirect(http.StatusFound, "")
}

func delete(ctx *gin.Context) {
	id := ctx.PostForm("id")
	var alertType AlertType
	db.First(&alertType, id)

	db.Delete(&alertType)

	ctx.Redirect(http.StatusFound, "")
}

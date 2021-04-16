package main

import (
	"flag"
	"html"
	"log"
	"net"
	"net/http"
	"os"

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
	Username   string `yaml:"username"`
	WebhookURL string `yaml:"webhookURL"`
	DBFilePath string `yaml:"dbFilePath"`
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

type Webhook struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

func alert(ctx *gin.Context) {
	alertTypeID := ctx.PostForm("alertTypeID")
	var alertType AlertType
	db.First(&alertType, alertTypeID)

	if alertType.Message != "" {
		log.Println("webhook! ", alertType.Message)
		c := resty.New()

		isIncludeVia := ctx.PostForm("isIncludeVia")
		via := ""
		if isIncludeVia == "on" {
			addr, err := net.LookupAddr(ctx.ClientIP())
			if err != nil {
				via = "\n (via " + ctx.ClientIP() + ")"
			} else {
				via = "\n (via " + ctx.ClientIP() + " -> " + addr[0] + ")"
			}
		}

		b := Webhook{
			Username: config.Username,
			Text:     "その話題..." + alertType.Message + "かも..." + via,
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

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/gin-gonic/gin"
	"github.com/teris-io/shortid"
	"google.golang.org/api/option"
)

var baseUrl = "http://localhost:5000/"

func main() {
	r := gin.Default()

	r.POST("/shorten", shorten)
	r.GET("/:code", redirect)
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{"message": "Welcome to shrtn url"})
	})

	r.Run(":5000")
}

type postBody struct {
	LongUlr string `json:"longUrl"`
}

func shorten(c *gin.Context) {
	var body postBody
	ctx := context.Background()
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, urlErr := url.ParseRequestURI(body.LongUlr)
	if urlErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": urlErr.Error()})
		return
	}
	client := dbInitx(ctx)
	ref := client.Collection("urls")

	urlCode, err := shortid.Generate()
	if err != nil {
		log.Fatal(err)
		return
	}
	dsnap, err := ref.Doc(urlCode).Get(ctx)
	if err != nil {
		log.Print("empty")
	}
	if dsnap.Exists() {
		url := dsnap.Data()
		c.JSON(http.StatusAccepted, url)
		return
	}
	var shortUrl = baseUrl + urlCode
	now := time.Now()
	expire := now.AddDate(0, 0, 7)
	doc := make(map[string]interface{})
	doc["urlCode"] = urlCode
	doc["longUrl"] = body.LongUlr
	doc["shortUrl"] = shortUrl
	doc["postedDate"] = now
	doc["expiresDate"] = expire
	_, er := ref.Doc(urlCode).Set(ctx, doc)

	if er != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": er.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"newUrl": shortUrl, "expires": expire.String(), "db_id": urlCode})
}

func redirect(c *gin.Context) {
	code := c.Param("code")
	ctx := context.Background()
	client := dbInitx(ctx)
	ref := client.Collection("urls")
	dsnap, err := ref.Doc(code).Get(ctx)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if dsnap.Exists() {
		urlData := dsnap.Data()
		var longUrl string = fmt.Sprint(urlData["longUrl"])
		log.Print(longUrl)
		c.Redirect(http.StatusPermanentRedirect, longUrl)
		return
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Url not Found"})
		return
	}
}

func dbInitx(context context.Context) (clnt *firestore.Client) {
	sa := option.WithCredentialsFile("firebase.json")
	app, err := firebase.NewApp(context, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(context)
	if err != nil {
		log.Fatalln(err)
	}
	return client
}

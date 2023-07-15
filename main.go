package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	goscrapper "github.com/AhmadWaleed/go-scrapper"
	"github.com/apex/gateway"

	//"github.com/aws/aws-lambda-go/lambda"
	"github.com/gin-gonic/gin"
)

func main() {

	if inLambda() {
		fmt.Println("running aws lambda in aws")
		log.Fatal(gateway.ListenAndServe(":8080", setupRouter()))
	} else {
		fmt.Println("running aws lambda in local")
		log.Fatal(http.ListenAndServe(":8080", setupRouter()))
	}

}

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.GET("/", runGet)
	return router
}

func runGet(c *gin.Context) {
	defer returnError(c)

	URL := c.Request.URL.Query().Get("url") // example usage - http://localhost:8080/?url=http[s]://example.com/
	if URL == "" {
		panic("Empty URL")
	}

	web, err := GetContent(URL)
	if err != nil {
		panic(err.Error())
	}
	links := CollectLinks(web, URL)

	mails, err := web.Emails()
	if err != nil {
		panic(err.Error())
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"links":  links,
		"mails":  mails,
	})
}

func inLambda() bool {
	if lambdaTaskRoot := os.Getenv("LAMBDA_TASK_ROOT"); lambdaTaskRoot != "" {
		return true
	}
	return false
}

func returnError(c *gin.Context) {
	if err := recover(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err,
		})
	}
}

func CollectLinks(web *goscrapper.Web, url string) []string {
	links := web.LinksWithDetails()
	filteredLinks := []string{}

	for _, link := range links {
		if len(link) > 0 {
			isInternal, filteredUrl := UrlFilter(link["url"].(string), url)
			if (isInternal == true) && (filteredUrl != nil) {
				filteredLinks = append(filteredLinks, strings.TrimSpace(filteredUrl.String()))
			}
		}
	}

	return filteredLinks
}

func UrlFilter(link string, URL string) (bool, *url.URL) {
	link = strings.TrimSpace(link)
	if (strings.HasSuffix(link, ".pdf")) ||
		(strings.HasSuffix(link, ".doc")) ||
		(strings.HasSuffix(link, ".docx")) ||
		(strings.HasSuffix(link, ".xls")) ||
		(strings.HasSuffix(link, ".xlsx")) ||
		(strings.HasSuffix(link, ".rar")) ||
		(strings.HasSuffix(link, ".zip")) ||
		(strings.HasSuffix(link, ".7zip")) ||
		(strings.HasSuffix(link, ".tar")) ||
		(strings.HasSuffix(link, ".tar.gz")) ||
		(strings.HasSuffix(link, ".jpg")) ||
		(strings.HasSuffix(link, ".png")) ||
		(strings.HasSuffix(link, ".gif")) ||
		(strings.HasSuffix(link, ".jpeg")) {
		return false, nil
	}

	l, err := url.Parse(link)
	if HasError(err) {
		return false, nil
	}

	base, err := url.Parse(strings.TrimSpace(URL))
	if HasError(err) {
		return false, nil
	}

	if l.Hostname() != base.Hostname() && (len(l.Hostname()) > 0) {
		return false, nil
	}

	processedLink := base.ResolveReference(l)

	if (processedLink.Hostname() == base.Hostname()) && (processedLink.String() != base.String()) {
		return true, processedLink
	}
	return false, nil
}

func GetContent(url string) (*goscrapper.Web, error) {
	counter := 1

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*5000)
	defer cancel()

	web, err := goscrapper.NewContextScrapper(ctx, url)
	if (err != nil) && (strings.Contains(err.Error(), "404 Not Found")) {
		return nil, err
	}
	for (counter < 5) && (err != nil) {
		web, err = goscrapper.NewContextScrapper(ctx, url)
		counter += 1
		log.Printf("Reattempt to get content of \"%s\" number: %d \n", url, counter)
	}
	return web, err
}

func HasError(err error) bool {
	if err != nil {
		if !inLambda() {
			log.Println(err)
		}
		return true
	}
	return false
}

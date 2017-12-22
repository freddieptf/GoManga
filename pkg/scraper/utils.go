package mangascraper

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	transport *http.Transport = &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		ResponseHeaderTimeout: 20 * time.Second,
	}
	client *http.Client = &http.Client{Transport: transport, Timeout: 40 * time.Second}
)

func makeDocRequest(url string) (*goquery.Document, error) {
	resp, err := client.Get(url)
	if err != nil {
		return &goquery.Document{}, err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return &goquery.Document{}, err
	}

	return doc, nil
}

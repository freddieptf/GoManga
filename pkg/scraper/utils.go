package scraper

import (
	"crypto/tls"
	"fmt"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sourcegraph/webloop"
	"golang.org/x/net/html"
	"net/http"
	"os"
	"runtime"
	"strings"
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

func makeDocRequestWebKit(url string) (*goquery.Document, error) {
	gtk.Init(nil)
	go func() {
		runtime.LockOSThread()
		gtk.Main()
	}()

	ctx := webloop.New()
	view := ctx.NewView()

	settings := view.Settings()
	settings.SetUserAgentWithApplicationDetails("Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:62.0)", "Firefox/62.0")

	defer view.Close()
	view.Open(url)
	err := view.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load URL: %s\n", err)
		return nil, err
	}
	res, err := view.EvaluateJavaScript("document.documentElement.innerHTML")
	if err != nil {
		fmt.Printf("damn, %s\n", err)
		return nil, err
	}
	node, err := html.Parse(strings.NewReader(res.(string)))
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	doc := goquery.NewDocumentFromNode(node)
	return doc, nil

}

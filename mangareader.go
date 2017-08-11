package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/go-playground/pool.v1"

	"github.com/PuerkitoBio/goquery"
)

type readerManga struct {
	MangaName *string
	Args      *[]int //chapters||volumes to download
	sourceUrl string
}

type readerChapter struct {
	sourceUrl  string
	mangaId    string
	chapterUrl string
	manga      string //name of the manga
	chapter    string
}

// GetFromReader get manga chapters from mangareader
// param n here is the number of active parallel downloads
func (d *readerManga) getChapters(n int) {
	results, err := d.search()
	if err != nil {
		log.Fatal(err)
	}

	match := getMatchFromSearchResults(results)
	*d.MangaName = match.manga

	downloader := chapterDownloader{}.init(n, len(*d.Args))

	for _, chapter := range *d.Args {
		downloader.queue(&readerChapter{
			sourceUrl:  d.sourceUrl,
			mangaId:    match.mangaID,
			manga:      *d.MangaName,
			chapter:    strconv.Itoa(chapter),
			chapterUrl: d.sourceUrl + match.mangaID + "/" + strconv.Itoa(chapter),
		})
	}

	for result := range downloader.startDownloads() { //receive results
		err, ok := result.(*pool.ErrRecovery)
		if ok { // there was some sort of panic that
			log.Println(err) // was recovered, in this scenario
			return
		}
		res := result.(string)
		fmt.Println("Download Successful: ", res)
	}

}

func (c *readerChapter) getMangaName() string {
	return c.manga
}

func (c *readerChapter) getChapterName() string {
	return c.chapter
}

//@TODO If a chapter doesn't exist onsite, return an error.
func (c *readerChapter) getChapter() error {
	var (
		urls    []string
		imgUrls []imgItem
		doc     *goquery.Document
		err     error
	)

	doc, err = goquery.NewDocument(c.chapterUrl)
	if err != nil {
		return err
	}

	//get the manga page urls
	doc.Find("div#selectpage > select#pageMenu > option").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Attr("value")
		url = c.sourceUrl + url
		urls = append(urls, url)
	})

	//go off and get the chapter title from the chapter listings
	titleChan := make(chan string)
	go func() {
		var title string
		d, e := goquery.NewDocument(c.sourceUrl + c.mangaId)
		if e != nil {
			log.Println(e)
			titleChan <- title
			return
		}
		d.Find("div#chapterlist > table#listing td").Has("div.chico_manga").EachWithBreak(func(i int, s *goquery.Selection) bool {
			link, _ := s.Find("a").Attr("href")
			ps := strings.Split(link, "/")
			if ps[2] == c.chapter {
				title = strings.Split(s.Text(), " : ")[1]
				return false
			}
			return true
		})
		titleChan <- title
	}()

	//scrape the manga pages for the image urls
	fmt.Printf("%v %v: Getting the chapter image urls\n", c.manga, c.chapter)

	imgItemChan := make(chan imgItem)
	for i, url := range urls {
		go func(i int, url string) {
			doc, err = goquery.NewDocument(url)
			if err != nil {
				log.Println(err)
				return
			}
			imgURL, _ := doc.Find("div#imgholder img").Attr("src")
			imgItemChan <- imgItem{ID: i, URL: imgURL}
		}(i, url)
	}

	for i := 0; i < len(urls); i++ {
		imgUrls = append(imgUrls, <-imgItemChan)
	}
	chapterTitle := <-titleChan
	chapterPath := filepath.Join(os.Getenv("HOME"), "Manga", "MangaReader", c.manga, c.chapter+": "+chapterTitle)
	err = os.MkdirAll(chapterPath, 0777)
	if err != nil {
		log.Println("Couldn't make directory ", chapterPath)
		return err
	}
	fmt.Printf("Downloading %s %s to %v \n", c.manga, c.chapter, chapterPath)
	ch := make(chan error)
	for _, item := range imgUrls {
		go func(item imgItem) {
			err = item.downloadImage(chapterPath)
			if err != nil {
				ch <- err
			}
			ch <- nil
		}(item)
	}
	for range imgUrls {
		err := <-ch
		if err != nil {
			os.RemoveAll(chapterPath)
			return err
		}
	}

	err = cbzify(chapterPath)
	if err != nil {
		fmt.Printf("Couldn't make chapter cbz: %v\n", err)
	}

	return nil
}

func (d *readerManga) getVolumes(n int) {}

func (download *readerManga) search() (map[int]searchResult, error) {
	doc, err := goquery.NewDocument(download.sourceUrl + "/alphabetical")
	if err != nil {
		return nil, err
	}
	var results = make(map[int]searchResult)
	//find possible matches in the site's manga list for the mangaName provided;
	doc.Find("ul.series_alpha > li > a").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(strings.ToLower(s.Text()), strings.ToLower(*download.MangaName)) {
			mid, _ := s.Attr("href")
			results[i] = searchResult{s.Text(), mid}
		}
	})

	if len(results) <= 0 {
		return nil, errors.New("found Zero results. Exiting")
	}

	return results, nil
}

package mangascraper

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	mangaReaderURL = "http://www.mangareader.net"
)

type ReaderManga struct {
	Manga
	Args *[]int //chapters||volumes to download
}

type readerChapter struct {
	mangaId    string
	chapterUrl string
	mangaName  string //name of the manga
	chapterId  string //the id, usually a number.
}

// Scrape manga chapters from mangareader
// params: n - the number of active parallel scrapers to use
func (d *ReaderManga) ScrapeChapters(n int) *chan ScrapeResult {

	scrapeJobChan := make(chan *scrapeJob, n)
	resultChan := make(chan ScrapeResult)

	startScrapers(n, scrapeJobChan, &resultChan)

	go func() {
		for _, chapter := range *d.Args {
			ch := &readerChapter{
				mangaId:    d.MangaID,
				mangaName:  d.MangaName,
				chapterId:  strconv.Itoa(chapter),
				chapterUrl: mangaReaderURL + d.MangaID + "/" + strconv.Itoa(chapter),
			}
			scrapeJobChan <- &scrapeJob{chapter: ch}
		}
	}()

	return &resultChan
}

func (c *readerChapter) getChapter() (Chapter, error) {
	var (
		sitePageUrls []string
		chapterPages []ChapterPage
		doc          *goquery.Document
		err          error
	)

	doc, err = makeDocRequest(c.chapterUrl)
	if err != nil {
		return Chapter{}, err
	}

	//get the manga page urls
	doc.Find("div#selectpage > select#pageMenu > option").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Attr("value")
		url = mangaReaderURL + url
		sitePageUrls = append(sitePageUrls, url)
	})

	//go off and get the chapter title from the chapter listings
	titleChan := make(chan string)
	go func() {
		var title string
		d, e := makeDocRequest(mangaReaderURL + c.mangaId)
		if e != nil {
			log.Println(e)
			titleChan <- title
			return
		}
		d.Find("div#chapterlist > table#listing td").Has("div.chico_manga").EachWithBreak(func(i int, s *goquery.Selection) bool {
			link, _ := s.Find("a").Attr("href")
			ps := strings.Split(link, "/")
			if ps[2] == c.chapterId {
				title = strings.Split(s.Text(), " : ")[1]
				return false
			}
			return true
		})
		titleChan <- title
	}()

	//scrape the manga pages for the image urls
	log.Printf("%v %v: Getting the chapter image urls\n", c.mangaName, c.chapterId)

	pageChan := make(chan ChapterPage)
	for i, url := range sitePageUrls {
		go func(i int, url string) {
			doc, err = makeDocRequest(url)
			if err != nil {
				log.Println(err)
				return
			}
			imgURL, exists := doc.Find("div#imgholder img").Attr("src")
			if !exists {
				log.Println("Doesn't exist")
				return
			}
			pageChan <- ChapterPage{Page: i, Url: imgURL}
		}(i, url)
	}

	for i := 0; i < len(sitePageUrls); i++ {
		chapterPages = append(chapterPages, <-pageChan)
	}

	chapterTitle := <-titleChan
	chapter := Chapter{
		MangaName:    c.mangaName,
		ChapterTitle: fmt.Sprintf("%s: %s", c.chapterId, chapterTitle),
		ChapterPages: chapterPages,
		SourceName:   "MangaReader",
	}

	return chapter, nil
}

func (d *ReaderManga) ScrapeVolumes(n int) (VlmChapterCount, *chan ScrapeResult) {
	return VlmChapterCount{Err: errors.New("Unimplemented for MangaReader")}, nil
}

func (d *ReaderManga) SetManga(manga Manga) {
	d.Manga = manga
}

func (d *ReaderManga) SetArgs(args *[]int) {
	d.Args = args
}

func (d *ReaderManga) GetArgs() *[]int {
	return d.Args
}

func (download *ReaderManga) Search() ([]Manga, error) {
	doc, err := makeDocRequest(mangaReaderURL + "/alphabetical")
	results := []Manga{}
	if err != nil {
		return results, err
	}
	//find possible matches in the site's manga list for the mangaName provided;
	doc.Find("ul.series_alpha > li > a").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(strings.ToLower(s.Text()), strings.ToLower(download.MangaName)) {
			mid, _ := s.Attr("href")
			results = append(results, Manga{MangaName: s.Text(), MangaID: mid})
		}
	})

	if len(results) <= 0 {
		return results, errors.New("found Zero results. Exiting")
	}

	return results, nil
}

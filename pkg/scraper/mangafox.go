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
	foxURL             = "http://mangafox.me/"
	foxMangaListingURL = foxURL + "manga/"
)

type VlmChapterCount struct {
	ChapterCount int
	Err          error
}

type FoxManga struct {
	Manga
	Args *[]int //chapters||volumes to foxManga
}

type foxChapter struct {
	mangaId    string
	chapterUrl string
	mangaName  string //name of the manga
	chapterId  string

	volume    string            //what volume the chapter belongs to - used when scraping volumes
	volumeDoc *goquery.Document // volume doc if any so we don't have to get the doc again
}

//GetFromFox gets manga chapters from mangafox
func (d *FoxManga) ScrapeChapters(n int) *chan ScrapeResult {
	scrapeJobChan := make(chan *scrapeJob, n)
	resultChan := make(chan ScrapeResult)

	startScrapers(n, scrapeJobChan, &resultChan)

	go func() {
		for _, chapter := range *d.Args {
			ch := &foxChapter{
				chapterUrl: d.MangaID,
				mangaName:  d.MangaName,
				chapterId:  strconv.Itoa(chapter),
			}
			scrapeJobChan <- &scrapeJob{chapter: ch}
		}
	}()

	return &resultChan

}

//GetVolumeFromFox gets manga volumes from Mangafox
func (d *FoxManga) ScrapeVolumes(n int) (VlmChapterCount, *chan ScrapeResult) {
	scrapeJobChan := make(chan *scrapeJob, n)
	resultChan := make(chan ScrapeResult)
	vlmChapterCount := VlmChapterCount{ChapterCount: 0}

	doc, err := makeDocRequest(d.MangaID)
	if err != nil {
		log.Println(err)
		vlmChapterCount.Err = err
		return vlmChapterCount, nil
	}

	volumeMap, err := findFoxVolumes(doc, d)
	if err != nil {
		vlmChapterCount.Err = err
		return vlmChapterCount, nil
	}

	for _, chapters := range volumeMap {
		vlmChapterCount.ChapterCount += len(chapters)
	}

	startScrapers(n, scrapeJobChan, &resultChan)

	go func() {
		for volumeTitle, chapters := range volumeMap {
			for _, chapter := range chapters {
				ch := &foxChapter{
					mangaName: d.MangaName,
					chapterId: chapter,
					volume:    volumeTitle,
					volumeDoc: doc,
				}
				scrapeJobChan <- &scrapeJob{chapter: ch}
			}
		}
	}()

	return vlmChapterCount, &resultChan
}

func findFoxVolumes(doc *goquery.Document, d *FoxManga) (map[string][]string, error) {
	volumes := make(map[string][]string)
	var errr error
	doc.Find("div.slide").Each(func(i int, s *goquery.Selection) {
		for _, v := range *d.Args {
			st := strings.Split(s.Find("h3.volume").Text(), " Chapter ")
			vi, err := strconv.Atoi(strings.Split(st[0], "Volume ")[1])
			if err != nil {
				log.Println(err)
				errr = err
				return
			}
			if v == vi {
				volumes[st[0]] = []string{}
				as := s.Next().First().Find("li a.tips")
				for i := as.Size() - 1; i >= 0; i-- {
					a := as.Eq(i)
					volumes[st[0]] = append(volumes[st[0]], strings.Split(a.Text(), d.MangaName+" ")[1])
				}
			}
		}
	})
	return volumes, errr
}

func (c *foxChapter) getChapter() (Chapter, error) {
	var err error
	var doc *goquery.Document

	if c.volumeDoc == nil {
		doc, err = makeDocRequest(c.chapterUrl) //open the manga's page on mangafox
		if err != nil {
			return Chapter{}, err
		}
	} else {
		doc = c.volumeDoc
	}

	var page1 string
	var urls []string

	var chapterPages []ChapterPage

	doc.Find("ul.chlist li").EachWithBreak(func(i int, s *goquery.Selection) bool {
		chID := strings.TrimPrefix(s.Find("a").Text(), c.mangaName+" ")
		if c.chapterId == chID { // search for the matching chapter in the manga's chapter catalogue
			page1, _ = s.Find("a").Last().Attr("href")
			return false
		}
		return true
	})

	baseURL := fmt.Sprintf("http:%s", strings.TrimSuffix(page1, "1.html"))
	doc, err = makeDocRequest(baseURL)
	if err != nil {
		return Chapter{}, err
	}

	titleChan := make(chan string)
	go func(doc *goquery.Document) { //goroutine to get the chapter's title
		titleChan <- strings.Split(doc.Find("div#tip").Find("strong").First().Text(), ": ")[1]
	}(doc)

	//get the num of chapter pages so we can build all the page urls
	doc.Find("div#top_center_bar select.m option").Each(func(i int, s *goquery.Selection) {
		urlID := s.Text()                          //get chapter page id in the select option..
		urls = append(urls, baseURL+urlID+".html") //"build" page urls and add them to our urls slice
	})

	if len(urls) == 0 { //if zero something went wrong
		return Chapter{}, errors.New("OOPS. CAN'T GET DIS: " + c.chapterId)
	}

	fmt.Printf("%v %v: Getting the chapter image urls\n", c.mangaName, c.chapterId)

	for i, url := range urls[:len(urls)-1] { //range over the slice..leave the last item out cause it's mostly always not valid
		doc, err = makeDocRequest(url) //open a chapter page
		if err != nil {
			log.Println(err)
		} else {
			imgURL, _ := doc.Find("div.read_img img").Attr("src") //get the image url
			chapterPages = append(chapterPages, ChapterPage{Url: imgURL, Page: i})
		}
	}

	chapterTitle := <-titleChan
	chapter := Chapter{
		MangaName:    c.mangaName,
		ChapterTitle: fmt.Sprintf("%s: %s", c.chapterId, chapterTitle),
		VolumeTitle:  c.volume,
		ChapterPages: chapterPages,
		SourceName:   "MangaFox",
	}

	return chapter, nil
}

func (d *FoxManga) SetManga(manga Manga) {
	d.Manga = manga
}

func (d *FoxManga) SetArgs(args *[]int) {
	d.Args = args
}

func (d *FoxManga) GetArgs() *[]int {
	return d.Args
}

//search the mangafox mangalist given a manga name string, returns the collection of results
func (foxManga *FoxManga) Search() ([]Manga, error) {
	doc, err := makeDocRequest(foxMangaListingURL)
	results := []Manga{}
	if err != nil {
		return results, err
	}

	doc.Find("div.manga_list li > a").Each(func(i int, s *goquery.Selection) { //go through the mangalist until we find matches
		if strings.Contains(strings.ToLower(s.Text()), strings.ToLower(foxManga.MangaName)) {
			mid, _ := s.Attr("href")
			results = append(results, Manga{MangaName: s.Text(), MangaID: fmt.Sprintf("http:%s", mid)})
		}
	})

	if len(results) <= 0 {
		return results, errors.New("found Zero results. Exiting")
	}

	return results, nil
}

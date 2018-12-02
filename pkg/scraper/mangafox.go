package scraper

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	foxURL = "http://fanfox.net"
)

type FoxManga struct{}

// func findFoxVolumes(doc *goquery.Document, d *FoxManga) (map[string][]string, error) {
// 	volumes := make(map[string][]string)
// 	var errr error
// 	doc.Find("div.slide").Each(func(i int, s *goquery.Selection) {
// 		for _, v := range *d.Args {
// 			st := strings.Split(s.Find("h3.volume").Text(), " Chapter ")
// 			vi, err := strconv.Atoi(strings.Split(st[0], "Volume ")[1])
// 			if err != nil {
// 				log.Println(err)
// 				errr = err
// 				return
// 			}
// 			if v == vi {
// 				volumes[st[0]] = []string{}
// 				as := s.Next().First().Find("li a.tips")
// 				for i := as.Size() - 1; i >= 0; i-- {
// 					a := as.Eq(i)
// 					volumes[st[0]] = append(volumes[st[0]], strings.Split(a.Text(), d.MangaName+" ")[1])
// 				}
// 			}
// 		}
// 	})
// 	return volumes, errr
// }

type mangaDetails struct {
	name        string
	description string
}

// get the manga details from the manga page
func getMangaDetails(doc *goquery.Document) mangaDetails {
	details := mangaDetails{}
	if doc == nil {
		return details
	}
	infoDiv := doc.Find("div.detail-info-right").First()
	details.name = infoDiv.Find("span.detail-info-right-title-font").Text()
	details.description = infoDiv.Find("p.fullcontent").Text()
	return details
}

// get the chapter url and title from the chapter listing on the manga page
func getChapterUrlFromListing(chapterID string, doc *goquery.Document) (chapterUrl, chapterTitle string) {
	doc.Find("ul.detail-main-list li a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		chUrl, exists := s.Attr("href")
		if !exists {
			return true // continue the loop
		}
		paths := strings.Split(chUrl, "/")
		chapterPath := paths[len(paths)-2]
		chID, err := strconv.ParseFloat(strings.TrimPrefix(chapterPath, "c"), 32)
		if err != nil {
			log.Printf("couldn't convert %s to valid chapter num: %v\n", chapterPath, err)
			return true // continue the loop
		}
		if inChID, err := strconv.ParseFloat(chapterID, 32); err == nil { // search for the matching chapter in the manga's chapter catalogue
			if inChID == chID {
				chapterUrl = fmt.Sprintf("%s%s", foxURL, chUrl)
				chapterTitle = s.Find("p.title1").Text()
				return false // break out of loop
			}
		}
		return true
	})
	return
}

// get all the chapter page urls from the select component on the chapter page
func getFoxChPageUrls(doc *goquery.Document) (chapterPageUrls []string) {
	lastPageSelection := doc.Find("div.pager-list-left a").Eq(-2)
	lastPage, _ := strconv.Atoi(lastPageSelection.AttrOr("data-page", "0")) // potential
	if lastPage == 0 {
		return
	}
	chapterID := strings.TrimSuffix(lastPageSelection.AttrOr("href", "none"), fmt.Sprintf("%d.html", lastPage)) // bugs
	for i := 1; i <= lastPage; i++ {
		chapterPageUrls = append(chapterPageUrls, fmt.Sprintf("%s%s%d.html", foxURL, chapterID, i))
	}
	return
}

// get the image url from a chapter page
func getFoxChPageImgUrl(chapterPageUrl string) (imgURL string) {
	doc, err := makeDocRequestWebKit(chapterPageUrl)
	if err != nil {
		log.Println(err)
		return
	}
	imgURL = doc.Find("img.reader-main-img").AttrOr("src", "")
	if _, err := url.ParseRequestURI(imgURL); err != nil {
		log.Printf("couldn't get chapter image url from %s : %v\n", chapterPageUrl, err)
	}
	return
}

func (foxManga *FoxManga) GetChapter(mangaID, chapterID string) (Chapter, error) {

	doc, err := makeDocRequest(fmt.Sprintf("%s/%s", foxURL, mangaID))
	if err != nil {
		log.Fatal(err)
	}

	mangaDetailsChan := make(chan mangaDetails)
	go func(doc *goquery.Document) {
		mangaDetailsChan <- getMangaDetails(doc)
	}(doc)

	chapterUrl, chapterTitle := getChapterUrlFromListing(chapterID, doc)

	doc, err = makeDocRequest(chapterUrl) // open the chapter page
	if err != nil {
		log.Printf("couldn't open chapter page %v\n", err)
		return Chapter{}, err
	}
	chapterPageUrls := getFoxChPageUrls(doc)

	if len(chapterPageUrls) == 0 { //if zero something went wrong
		return Chapter{}, errors.New("OOPS. CAN'T GET DIS: " + chapterID)
	}

	var chapterPages []ChapterPage

	for i, url := range chapterPageUrls[:len(chapterPageUrls)-1] { //range over the slice..leave the last item out cause it's mostly always not valid
		chapterPages = append(chapterPages, ChapterPage{Url: getFoxChPageImgUrl(url), Page: i})
	}

	chapter := Chapter{
		MangaName:    (<-mangaDetailsChan).name,
		ChapterTitle: fmt.Sprintf("%s: %s", chapterID, chapterTitle),
		ChapterPages: chapterPages,
		SourceName:   "MangaFox",
	}

	return chapter, nil
}

// search the mangafox mangalist given a manga name string, returns the collection of results
func (foxManga *FoxManga) Search(mangaName string) ([]Manga, error) {
	searchUrl := fmt.Sprintf("%s/search?name=%s", foxURL, mangaName)
	doc, err := makeDocRequest(searchUrl)
	results := []Manga{}
	if err != nil {
		return results, err
	}

	doc.Find("ul.manga-list-4-list li > a").Each(func(i int, s *goquery.Selection) { //go through the mangalist until we find matches
		if strings.Contains(strings.ToLower(s.AttrOr("title", "")), strings.ToLower(mangaName)) {
			mid, _ := s.Attr("href")
			mid = strings.TrimPrefix(mid, "/")
			results = append(results, Manga{MangaName: s.AttrOr("title", mangaName), MangaID: mid})
		}
	})

	if len(results) <= 0 {
		return results, errors.New("found Zero results. Exiting")
	}

	return results, nil
}

package scraper

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	foxURL             = "http://fanfox.net/"
	foxMangaListingURL = foxURL + "manga/"
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
	doc.Find("head meta").Each(func(idx int, s *goquery.Selection) {
		switch s.AttrOr("property", "none") {
		case "og:title":
			{
				val := s.AttrOr("content", "title not found Manga")
				details.name = strings.TrimSuffix(val, " Manga")
			}
		case "og:description":
			{
				val := s.AttrOr("content", "no description")
				details.description = val
			}
		}
	})
	return details
}

// get the chapter url and title from the chapter listing on the manga page
func getChapterUrlFromListing(chapterID string, doc *goquery.Document) (chapterUrl, chapterTitle string) {
	doc.Find("ul.chlist li").EachWithBreak(func(i int, s *goquery.Selection) bool {
		chID := strings.Split(s.Find("a").Text(), " ")[1]
		if chapterID == chID { // search for the matching chapter in the manga's chapter catalogue
			chapterUrl, _ = s.Find("a").Last().Attr("href")
			chapterUrl = fmt.Sprintf("http:%s", strings.TrimSuffix(chapterUrl, "1.html"))

			chapterTitle = s.Find("span.title").First().Text()
			if chapterTitle == "" {
				chapterTitle = s.Find("a").Last().Text()
			}

			return false
		}
		return true
	})
	return
}

// get all the chapter page urls from the select component on the chapter page
func getFoxChPageUrls(doc *goquery.Document) (chapterPageUrls []string) {
	baseURL := ""
	doc.Find("head meta").EachWithBreak(func(idx int, s *goquery.Selection) bool {
		sText, exists := s.Attr("property")
		if exists && sText == "og:url" {
			baseURL = s.AttrOr("content", "")
			return false
		}
		return true
	})
	doc.Find("div#top_center_bar select.m option").Each(func(i int, s *goquery.Selection) {
		pageID := s.Text()                                                //get chapter page id in the select option..
		chapterPageUrls = append(chapterPageUrls, baseURL+pageID+".html") //"build" page urls and add them to our urls slice
	})
	return
}

// get the image url from a chapter page
func getFoxChPageImgUrl(chapterPageUrl string) (imgURL string) {
	doc, err := makeDocRequest(chapterPageUrl)
	if err != nil {
		log.Println(err)
		return
	}
	imgURL, _ = doc.Find("div.read_img img").Attr("src") //get the image url
	if imgURL == "" {
		log.Printf("weird..couldn't get chapter image url from %s\n", chapterPageUrl)
	}
	return
}

func (foxManga *FoxManga) GetChapter(mangaID, chapterID string) (Chapter, error) {

	doc, err := makeDocRequest(foxMangaListingURL + mangaID)
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
	doc, err := makeDocRequest(foxMangaListingURL)
	results := []Manga{}
	if err != nil {
		return results, err
	}

	doc.Find("div.manga_list li > a").Each(func(i int, s *goquery.Selection) { //go through the mangalist until we find matches
		if strings.Contains(strings.ToLower(s.Text()), strings.ToLower(mangaName)) {
			mid, _ := s.Attr("href")
			mUrl, _ := url.Parse(fmt.Sprintf("http:%s", mid))
			mid = strings.Split(strings.Trim(mUrl.Path, "/"), "/")[1]
			results = append(results, Manga{MangaName: s.Text(), MangaID: mid})
		}
	})

	if len(results) <= 0 {
		return results, errors.New("found Zero results. Exiting")
	}

	return results, nil
}

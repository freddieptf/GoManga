package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"gopkg.in/go-playground/pool.v1"
)

type volume struct {
	volume   string
	chapters []string
}

type foxManga struct {
	MangaName *string
	Args      *[]int //chapters||volumes to download
	sourceUrl string
}

type foxChapter struct {
	sourceUrl  string
	mangaId    string
	chapterUrl string
	manga      string //name of the manga
	chapter    string
	volume     string            //what volume the chapter belongs to, optional really
	volumeDoc  *goquery.Document // volume doc if any so we don't have to get the doc again
}

//GetFromFox gets manga chapters from mangafox
func (d *foxManga) getChapters(n int) {
	results, err := d.search()
	if err != nil {
		log.Fatal(err)
	}

	match := getMatchFromSearchResults(results)

	downloader := chapterDownloader{}.init(n, len(*d.Args))

	for _, chapter := range *d.Args {
		downloader.queue(&foxChapter{
			chapterUrl: match.mangaID,
			manga:      match.manga,
			chapter:    strconv.Itoa(chapter),
		})
	}

	for result := range downloader.startDownloads() {
		err, ok := result.(*pool.ErrRecovery)
		if ok { // there was some sort of panic that
			fmt.Println(err) // was recovered, in this scenario
			return
		}
		res := result.(string)
		fmt.Println("Download Successful: ", res)
	}

}

//GetVolumeFromFox gets manga volumes from Mangafox
func (d *foxManga) getVolumes(n int) {
	results, err := d.search()
	if err != nil {
		log.Fatal(err)
	}

	match := getMatchFromSearchResults(results)

	doc, err := goquery.NewDocument(match.mangaID)
	if err != nil {
		log.Println(err)
		return
	}

	*d.MangaName = match.manga
	volumes := findFoxVolumes(doc, d)

	downloader := chapterDownloader{}.init(n, len(*d.Args))

	for i := len(volumes) - 1; i >= 0; i-- { //reverse the order since the older volumes are at the end...older first
		for _, chapter := range volumes[i].chapters {
			downloader.queue(&foxChapter{
				manga:     *d.MangaName,
				chapter:   chapter,
				volume:    volumes[i].volume,
				volumeDoc: doc,
			})
		}
	}

	for result := range downloader.startDownloads() {
		err, ok := result.(*pool.ErrRecovery)
		if ok { // there was some sort of panic that
			log.Println(err) // was recovered, in this scenario
			return
		}
		res := result.(string)
		fmt.Println("Download Successful: ", res)
	}

}

func findFoxVolumes(doc *goquery.Document, d *foxManga) []volume {
	var vols []volume
	doc.Find("div.slide").Each(func(i int, s *goquery.Selection) {
		for _, v := range *d.Args {
			st := strings.Split(s.Find("h3.volume").Text(), " Chapter ")
			vi, err := strconv.Atoi(strings.Split(st[0], "Volume ")[1])
			if err != nil {
				log.Printf("%v\n", err)
			}
			if v == vi {
				var vol volume
				vol.volume = st[0]
				as := s.Next().First().Find("li a.tips")
				for i := as.Size() - 1; i >= 0; i-- { //get oldest chapter to newest
					a := as.Eq(i)
					vol.chapters = append(vol.chapters, strings.Split(a.Text(), *d.MangaName+" ")[1])
				}
				vols = append(vols, vol)
			}
		}
	})
	return vols
}

func (c *foxChapter) getMangaName() string {
	return c.manga
}

func (c *foxChapter) getChapterName() string {
	return c.chapter
}

func (c *foxChapter) getChapter() error {
	var err error
	var doc *goquery.Document

	if c.volumeDoc == nil {
		doc, err = goquery.NewDocument(c.chapterUrl) //open the manga's page on mangafox
		if err != nil {
			return err
		}
	} else {
		doc = c.volumeDoc
	}

	var page1 string
	var urls []string
	var imgUrls []imgItem

	doc.Find("ul.chlist li").EachWithBreak(func(i int, s *goquery.Selection) bool {
		chID := strings.TrimPrefix(s.Find("a").Text(), c.manga+" ")
		if c.chapter == chID { // search for the matching chapter in the manga's chapter catalogue
			page1, _ = s.Find("a").Last().Attr("href")
			return false
		}
		return true
	})

	baseURL := strings.TrimSuffix(page1, "1.html")
	doc, err = goquery.NewDocument(baseURL) //get the chapter's page
	if err != nil {
		return err
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
		return errors.New("OOPS. CAN'T GET DIS: " + c.chapter)
	}

	fmt.Printf("%v %v: Getting the chapter image urls\n", c.manga, c.chapter)
	imgItemChan := make(chan imgItem)
	var wg sync.WaitGroup
	for i, url := range urls[:len(urls)-1] { //range over the slice..leave the last item out cause it's mostly always not valid
		wg.Add(1)
		go func(i int, url string) {
			doc, err = goquery.NewDocument(url) //open a chapter page
			if err != nil {
				log.Println(err)
				return
			}
			imgURL, _ := doc.Find("div.read_img img").Attr("src") //get the image url
			wg.Done()
			imgItemChan <- imgItem{URL: imgURL, ID: i}
		}(i, url)
		wg.Wait()
	}

	for i := 0; i < len(urls)-1; i++ {
		imgUrls = append(imgUrls, <-imgItemChan) //get dem image urls..append them to the slice
	}

	var chapterPath string
	if c.volume == "" {
		chapterPath = filepath.Join(os.Getenv("HOME"), "Manga", "MangaFox", c.manga, c.chapter+": "+<-titleChan)
	} else {
		chapterPath = filepath.Join(os.Getenv("HOME"), "Manga", "MangaFox", c.manga, c.volume, c.chapter+": "+<-titleChan)
	}
	err = os.MkdirAll(chapterPath, 0777)
	if err != nil {
		log.Fatal("Couldn't make directory ", err)
	}
	fmt.Printf("Downloading %s %s to %v: \n", c.manga, c.chapter, chapterPath)
	ch := make(chan error)
	for _, item := range imgUrls {
		go func(item imgItem) {
			err = item.downloadImage(chapterPath)
			if err != nil {
				ch <- err //send error if any while downloading an image
			}
			ch <- nil
		}(item)
	}

	for range imgUrls {
		err := <-ch //receive the error if any
		if err != nil {
			os.RemoveAll(chapterPath) //delete the whole chapter if one img download failed..bad but for now...meh
			return err                //return error and exit
		}
	}

	err = cbzify(chapterPath)
	if err != nil {
		fmt.Printf("Couldn't make chapter cbz: %v", err)
	}

	return nil
}

//search the mangafox mangalist given a manga name string, returns the collection of results
func (download *foxManga) search() (map[int]searchResult, error) {
	doc, err := goquery.NewDocument(download.sourceUrl + "manga/")
	if err != nil {
		return nil, err
	}

	var results = make(map[int]searchResult)
	doc.Find("div.manga_list li > a").Each(func(i int, s *goquery.Selection) { //go through the mangalist until we find matches
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

package main

import (
	"bufio"
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

//GetFromFox gets manga chapters from mangafox
func (d *foxManga) getChapters(n int) {
	results, err := d.search()
	if err != nil {
		log.Fatal(err)
	}

	match := getMatchFromSearchResults(results)

	p := pool.NewPool(n, len(*d.Args))

	fn := func(job *pool.Job) {
		e := job.Params()[0].(*chapterDownload).getChapterFromFox(nil, "") //pass empty cause it's not a volume download..nil cause we aren't caching none
		if e != nil {
			fmt.Printf("Download Failed: %v chapter %v (%v)\n",
				job.Params()[0].(*chapterDownload).manga, job.Params()[0].(*chapterDownload).chapter, e)
			return
		}
		job.Return(job.Params()[0].(*chapterDownload).manga + " " + job.Params()[0].(*chapterDownload).chapter)
	}

	for _, chapter := range *d.Args {
		p.Queue(fn, &chapterDownload{
			chapterUrl: match.mangaID,
			manga:      match.manga,
			chapter:    strconv.Itoa(chapter),
		})
	}

	for result := range p.Results() {
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

	var volumes []volume
	doc.Find("div.slide").Each(func(i int, s *goquery.Selection) {
		for _, v := range *d.Args {
			st := strings.Split(s.Find("h3.volume").Text(), " Chapter ")
			vi, err := strconv.Atoi(strings.Split(st[0], "Volume ")[1])
			if err != nil {
				return
			}
			if v == vi {
				var vol volume
				vol.volume = st[0]
				as := s.Next().First().Find("li a.tips")
				for i := as.Size() - 1; i >= 0; i-- { //get oldest chapter to newest
					a := as.Eq(i)
					vol.chapters = append(vol.chapters, strings.Split(a.Text(), *d.MangaName+" ")[1])
				}
				volumes = append(volumes, vol)
			}
		}
	})

	p := pool.NewPool(n, len(*d.Args))
	fn := func(job *pool.Job) {
		e := job.Params()[0].(*chapterDownload).getChapterFromFox(job.Params()[1].(*goquery.Document), //pass the doc..caching blah blah
			job.Params()[0].(*chapterDownload).volume) //pass what volume we're downloading from
		if e != nil {
			fmt.Printf("Download Failed: %v chapter %v (%v)\n",
				job.Params()[0].(*chapterDownload).manga, job.Params()[0].(*chapterDownload).chapter, e)
			return
		}
		job.Return(job.Params()[0].(*chapterDownload).manga + " " + job.Params()[0].(*chapterDownload).chapter)
	}

	for i := len(volumes) - 1; i >= 0; i-- { //reverse the order since the older volumes are at the end...older first
		for _, chapter := range volumes[i].chapters {
			p.Queue(fn, &chapterDownload{
				manga:   *d.MangaName,
				chapter: chapter,
				volume:  volumes[i].volume,
			}, doc)
		}
	}

	for result := range p.Results() {
		// err, ok := result.(*pool.ErrRecovery)
		// if ok { // there was some sort of panic that
		// 	log.Println(err) // was recovered, in this scenario
		// 	return
		// }
		res := result.(string)
		fmt.Println("Download Successful: ", res)
	}

}

//download a chapter from mangafox...
//@param volume used when creating the manga directories
//@param doc passed from GetVolumeFromFox so we don't create a new one
func (c *chapterDownload) getChapterFromFox(doc *goquery.Document, volume string) error {
	var err error
	if doc == nil {
		doc, err = goquery.NewDocument(c.chapterUrl) //open the manga's page on mangafox
		if err != nil {
			return err
		}
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
	if volume == "" {
		chapterPath = filepath.Join(os.Getenv("HOME"), "Manga", "MangaFox", c.manga, c.chapter+": "+<-titleChan)
	} else {
		chapterPath = filepath.Join(os.Getenv("HOME"), "Manga", "MangaFox", c.manga, volume, c.chapter+": "+<-titleChan)
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

func getMatchFromSearchResults(results map[int]searchResult) searchResult {
	fmt.Printf("Id \t Manga\n")
	for i, m := range results {
		fmt.Printf("%d \t %s\n", i, m.manga)
	}

	myScanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Enter the id of the correct manga: ")
	var id int
	var err error
scanDem:
	for myScanner.Scan() {
		id, err = strconv.Atoi(myScanner.Text())
		if err != nil {
			fmt.Printf("Enter a valid Id, please: ")
			goto scanDem
		}
		break
	}
	//get the matching id
	match, exists := results[id] // mangafox has the manga url also in the catalogue so we use that
	if !exists {
		fmt.Printf("Insert one of the Ids in the results, please: ")
		goto scanDem
	}
	return match
}

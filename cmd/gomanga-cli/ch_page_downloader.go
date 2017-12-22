package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"

	scraper "github.com/freddieptf/manga-scraper/pkg/scraper"
)

type pageDownloadJob struct {
	path        *string
	chapterPage *scraper.ChapterPage
}

type ChapterPageDownloader struct {
	maxParallel         int
	pageDownloadJobChan chan pageDownloadJob
}

type pageDownloader struct {
	wg   *sync.WaitGroup
	quit chan bool
}

// we pass in the max number of parallel pageDownloaders we want at a time and return a pointer to the struct
func initChapterPageDownloader(n int) *ChapterPageDownloader {
	return &ChapterPageDownloader{
		maxParallel:         n,
		pageDownloadJobChan: make(chan pageDownloadJob, n),
	}
}

// create the number of pageDownloaders needed and get them waiting for any new downloads posted to the pageDownloadChan channel
func (chapterPageDownloader *ChapterPageDownloader) initWorkers(wg *sync.WaitGroup) {
	for i := 0; i < chapterPageDownloader.maxParallel; i++ {
		pageDownloader := &pageDownloader{quit: make(chan bool), wg: wg}
		pageDownloader.waitForWork(chapterPageDownloader.pageDownloadJobChan)
	}
}

// post a download to the pageDownloadJobChan channel. This will block until an idle pageDownloader is avalilabe to service the job
func (chapterPageDownloader *ChapterPageDownloader) submitPageForDownload(path *string, chapterPage scraper.ChapterPage) {
	chapterPageDownloader.pageDownloadJobChan <- pageDownloadJob{path: path, chapterPage: &chapterPage}
}

// start listening for any downloadJobs posted on the pageDownloadJob channel
func (pageDownloader *pageDownloader) waitForWork(jobChan chan pageDownloadJob) {
	go func() {
		for {
			select {
			case job := <-jobChan:
				err := DownloadChapterPage(job.path, job.chapterPage)
				if err != nil {
					log.Println(err)
				}
				pageDownloader.wg.Done()
			case <-pageDownloader.quit:
				return
			}
		}
	}()
}

// download a chapter page and save it to disk at the path provided. Returns error or nil if none occurs.
func DownloadChapterPage(path *string, chapterPage *scraper.ChapterPage) error {
	response, err := http.Get(chapterPage.Url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	imgPath := filepath.Join(*path, strconv.Itoa(chapterPage.Page)+".jpg")
	err = ioutil.WriteFile(imgPath, body, 0655)
	if err != nil {
		return err
	}
	return nil
}

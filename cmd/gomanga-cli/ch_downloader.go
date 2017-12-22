package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	scraper "github.com/freddieptf/manga-scraper/pkg/scraper"
)

var downloadJobChan chan scraper.Chapter

type chapterDownloader struct {
	id   int
	quit chan bool
	wg   *sync.WaitGroup
}

func newChapterDownloader(id int, wg *sync.WaitGroup) *chapterDownloader {
	return &chapterDownloader{
		id:   id,
		wg:   wg,
		quit: make(chan bool),
	}
}

// starts listening for jobs posted on the downloadJobChan channel
func (chDown *chapterDownloader) listen() {
	go func() {
		for {
			select {
			case chapter := <-downloadJobChan:
				log.Printf("downloader %d: Start new Download %s : vlm %s\n", chDown.id, chapter.ChapterTitle, chapter.VolumeTitle)
				path, err := DownloadChapter(&chapter)
				if err != nil {
					log.Printf("couldn't get %s : err %v\n", chapter.ChapterTitle, err)
				}
				log.Printf("downloader %d: Download done (%s): vlm %s\n", chDown.id, path, chapter.VolumeTitle)
				chDown.wg.Done()
			case <-chDown.quit:
				return
			}
		}
	}()
}

// params: 	n - the max active parallel downloads
//			total - the total no of downloads
//			resultsChan - the channel that provides the scrapeResult:{ either a chapter or an error }
func startDownloads(n, total int, resultsChan *chan scraper.ScrapeResult) {
	downloadJobChan = make(chan scraper.Chapter, n)
	var wg sync.WaitGroup

	// init our download workers right here boy
	for i := 0; i < n; i++ {
		chDownloader := newChapterDownloader(i+1, &wg)
		chDownloader.listen()
	}

	count := 0
	for {
		select {
		case scrapeResult := <-*resultsChan:
			count++
			wg.Add(1)
			if scrapeResult.Err != nil {
				log.Printf("Damn err, %v\n", scrapeResult.Err)
				wg.Done()
			} else {
				downloadJobChan <- scrapeResult.Chapter
			}
			if count == total { // if the no of results we've got == the total no of downloads
				wg.Wait() // then we wait for the downloads to finish
				return    // and exit
			}
		}
	}
}

func makeMangaSourceDir(sourceName string) (string, error) {
	sourceDir := filepath.Join(os.Getenv("HOME"), "Manga", sourceName)
	err := os.MkdirAll(sourceDir, 0777)
	if err != nil {
		return "", err
	}
	return sourceDir, nil
}

// downloads the chapter's pages to the chapter dir and returns its path on disk
func DownloadChapter(chapter *scraper.Chapter) (string, error) {
	sourceDirPath, err := makeMangaSourceDir(chapter.SourceName)
	if err != nil {
		log.Fatalf("couldn't make source %s library dir: err %v", chapter.SourceName, err)
	}

	var chapterPath string

	if chapter.VolumeTitle == "" {
		chapterPath = filepath.Join(sourceDirPath, chapter.MangaName, chapter.ChapterTitle)
	} else {
		chapterPath = filepath.Join(sourceDirPath, chapter.MangaName, chapter.VolumeTitle, chapter.ChapterTitle)
	}

	err = os.MkdirAll(chapterPath, 0777)
	if err != nil {
		log.Fatalf("Couldn't make %s directory: err %v", chapterPath, err)
	}

	fmt.Printf("Downloading %s %s to %v: \n", chapter.MangaName, chapter.ChapterTitle, chapterPath)

	chapterPageDownloader := initChapterPageDownloader(12) // set our max no of workers to 12
	var wg sync.WaitGroup
	chapterPageDownloader.initWorkers(&wg)
	for _, chapterPage := range chapter.ChapterPages {
		wg.Add(1)
		chapterPageDownloader.submitPageForDownload(&chapterPath, chapterPage)
	}
	wg.Wait() // wait for all the downloads to complete before exiting
	return chapterPath, nil
}

package cli

import (
	"fmt"
	"github.com/freddieptf/manga-scraper/pkg/scraper"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type chapterDownloader struct {
	id      int
	jobChan *chan scraper.Chapter
	quit    chan struct{}
	wg      *sync.WaitGroup
}

func newChapterDownloader(
	id int,
	wg *sync.WaitGroup,
	jobChan *chan scraper.Chapter) *chapterDownloader {

	return &chapterDownloader{
		id:      id,
		wg:      wg,
		jobChan: jobChan,
		quit:    make(chan struct{}),
	}
}

// starts listening for jobs posted on the downloadJobChan channel
func (chDown *chapterDownloader) listen() {
	go func() {
		for {
			select {
			case chapter := <-*chDown.jobChan:
				log.Printf("downloader %d: Start new Download %s : vlm %s\n", chDown.id, chapter.ChapterTitle, chapter.VolumeTitle)
				path, err := downloadChapter(&chapter)
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

func startDownloads(n int, wg *sync.WaitGroup, resultsChan *chan scraper.Chapter) {
	// init our download workers right here boy
	for i := 0; i < n; i++ {
		chDownloader := newChapterDownloader(i+1, wg, resultsChan)
		chDownloader.listen()
	}
}

// make the manga source dir under the apps root dir, i.e Manga in the home dir
// returns the path of the dir created
func createMangaSourceDir(sourceName string) (string, error) {
	sourceDir := filepath.Join(os.Getenv("HOME"), "Manga", sourceName)
	err := os.MkdirAll(sourceDir, 0777)
	if err != nil {
		return "", err
	}
	return sourceDir, nil
}

// create a chapter dir in the provided manga source dir path, return the its path
func createChapterDir(sourceDirPath *string, chapter *scraper.Chapter) (string, error) {
	var chapterPath string
	if chapter.VolumeTitle == "" {
		chapterPath = filepath.Join(*sourceDirPath, chapter.MangaName, chapter.ChapterTitle)
	} else {
		chapterPath = filepath.Join(*sourceDirPath, chapter.MangaName, chapter.VolumeTitle, chapter.ChapterTitle)
	}

	err := os.MkdirAll(chapterPath, 0777)
	if err != nil {
		return "", fmt.Errorf("Couldn't make %s directory: err %v", chapterPath, err)
	}

	return chapterPath, nil
}

// get image given the url, return a byte array to do whatever we want with it
func getImage(url string) (*[]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return &body, nil
}

func saveImageToDisk(path string, img *[]byte) error {
	return ioutil.WriteFile(path, *img, 0655)
}

func downloadChapter(
	chapter *scraper.Chapter) (string, error) {

	sourceDirPath, err := createMangaSourceDir(chapter.SourceName)
	if err != nil {
		log.Fatalf("couldn't make source %s library dir: err %v", chapter.SourceName, err)
	}

	chapterPath, err := createChapterDir(&sourceDirPath, chapter)
	if err != nil {
		log.Fatalf("could not create chapterDir %s\n", err)
	}

	fmt.Printf("downloading: %s\n", chapterPath)

	for _, chapterPage := range chapter.ChapterPages {
		pagePath := filepath.Join(chapterPath, strconv.Itoa(chapterPage.Page)+".jpg")
		img, err := getImage(chapterPage.Url)
		if err != nil {
			fmt.Printf("%v occured while getting %s\n", err, chapterPage.Url)
		} else {
			err = saveImageToDisk(pagePath, img)
			if err != nil {
				fmt.Printf("%v, couldn't save img to %s\n", err, pagePath)
			}
		}
	}

	return chapterPath, nil
}

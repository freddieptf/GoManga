package mangascraper

import (
	"log"
)

type scrapeJob struct {
	chapter sourceChapter
}

type scraper struct {
	jobChan           chan *scrapeJob
	scrapeResultsChan *chan ScrapeResult
	quitChan          chan bool
}

type ScrapeResult struct {
	Chapter Chapter
	Err     error
}

func startScrapers(maxScrapers int, jobChan chan *scrapeJob, resultChan *chan ScrapeResult) {
	for i := 0; i < maxScrapers; i++ {
		scraper := newScraper(jobChan, resultChan)
		scraper.starto()
	}
}

func newScraper(scrapeJobChan chan *scrapeJob, resultChan *chan ScrapeResult) *scraper {
	return &scraper{
		jobChan:           scrapeJobChan,
		scrapeResultsChan: resultChan,
		quitChan:          make(chan bool),
	}
}

// start listening for work
func (scraper *scraper) starto() {
	go func() {
		for {
			select {
			case job := <-scraper.jobChan:
				ch, err := job.chapter.getChapter()
				*scraper.scrapeResultsChan <- ScrapeResult{Chapter: ch, Err: err}
			case <-scraper.quitChan:
				log.Printf("scraper received quit signal")
				// return, stop listening for work
				return
			}
		}
	}()
}

// stop listening for work and stop when whatever is running finishes
func (scraper *scraper) korosu() {
	go func() {
		scraper.quitChan <- true
	}()
}

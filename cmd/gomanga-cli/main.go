package main

import (
	"fmt"
	"log"

	scraper "github.com/freddieptf/manga-scraper/pkg/scraper"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	foxFlag            = kingpin.Flag("mf", "use mangafox as source").Bool()
	readerFlag         = kingpin.Flag("mr", "use mangaReader as source").Bool()
	vlm                = kingpin.Flag("vlm", "use when you want to download a volume(s)").Bool()
	update             = kingpin.Flag("update", "use to update the manga in your lib to the latest chapter").Bool()
	maxActiveDownloads = kingpin.Flag("n", "set max number of concurrent downloads").Int()
	manga              = kingpin.Arg("manga", "The name of the manga").String()
	args               = kingpin.Arg("arguments",
		"chapters (volumes if --vlm is set) to download. Example format: 1 3 5-7 67 10-14").Strings()
)

func main() {
	kingpin.Parse()
	n := 1 //default num of maxActiveDownloads
	if *maxActiveDownloads != 0 {
		n = *maxActiveDownloads
	}

	if *foxFlag {
		n = 1 // so we don't hammer the mangafox site...we start getting errors if we set this any higher
		fmt.Println("Setting max active downloads to 1. " +
			"You will get errors or missing chapter images if you set it any higher with mangafox as source.")
	}

	var source scraper.MangaSource

	switch {
	case *foxFlag:
		source = &scraper.FoxManga{}
		source.SetManga(scraper.Manga{MangaName: *manga})
		source.SetArgs(getRange(args))
		if *vlm { //if we're downloading volumes
			getVolumes(n, source)
		} else {
			getChapters(n, source)

		}
	case *readerFlag:
		source = &scraper.ReaderManga{}
		source.SetManga(scraper.Manga{MangaName: *manga})
		source.SetArgs(getRange(args))
		getChapters(n, source)
	case *update:
		// updateMangaLib()
	default:
		fmt.Println("Default source used: MangaReader.")
		// source = &ReaderManga{
		// 	Args:      GetRange(args),
		// 	MangaName: manga,
		// }
		// source.GetChapters(n)
	}

}

func getChapters(n int, source scraper.MangaSource) {
	results, err := source.Search()
	if err != nil {
		log.Fatal(err)
	}
	result := getMatchFromSearchResults(results)
	source.SetManga(result)

	resultsChan := source.ScrapeChapters(n)
	startDownloads(n, len(*source.GetArgs()), resultsChan)
}

func getVolumes(n int, source scraper.MangaSource) {
	results, err := source.Search()
	if err != nil {
		log.Fatal(err)
	}
	result := getMatchFromSearchResults(results)
	source.SetManga(result)

	count, resultsChan := source.ScrapeVolumes(n)
	if count.Err != nil {
		log.Fatalf("Encountered an error while trying to scrape volumes : err %v\n", count.Err)
	}
	startDownloads(n, count.ChapterCount, resultsChan)
}

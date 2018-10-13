package main

import (
	"flag"
	"fmt"
	cli "github.com/freddieptf/manga-scraper/pkg/cli"
)

var (
	foxFlag            = flag.Bool("mf", false, "search mangafox for the manga")
	readerFlag         = flag.Bool("mr", false, "search mangaReader for the manga")
	vlm                = flag.Bool("vlm", false, "use with -mf when you want to download a volume(s)")
	update             = flag.Bool("update", false, "use to update the manga in your local library to the latest chapter")
	maxActiveDownloads = flag.Int("n", 1, "max number of concurrent downloads")
	manga              = flag.String("manga", "", "the name of the manga")
)

func main() {
	flag.Parse()
	args := flag.Args()

	if !*foxFlag && !*readerFlag {
		fmt.Println("No source was provided. See -h for usage. Using mangareader as default instead...")
		*readerFlag = true
	}

	// construct a config struct that we'll pass to cli.Run()..blocks until completion
	conf := cli.CliConf{
		IsSourceFox:           foxFlag,
		IsSourceRdr:           readerFlag,
		Vlms:                  vlm,
		MangaName:             manga,
		ChapterArgs:           &args,
		ParallelDownloadLimit: maxActiveDownloads,
	}

	cli.Run(conf)

}

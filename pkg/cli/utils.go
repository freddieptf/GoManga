package cli

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	scraper "github.com/freddieptf/manga-scraper/pkg/scraper"
)

func GetChapterRange(arg string, ran *[]int, entryMap *map[int]struct{}) {
	chs := strings.Split(arg, "-")
	if len(chs) > 1 {
		// exit early if neccessary
		if len(chs[0]) == 0 {
			if len(chs[1]) > 0 {
				chs[0] = chs[1]
			} else {
				return
			}
		} else if len(chs[1]) == 0 {
			if len(chs[0]) > 0 {
				chs[1] = chs[0]
			} else {
				return
			}
		}

		start, err := strconv.Atoi(chs[0])
		if err != nil {
			log.Fatalf("Could not convert %v to a valid chapter: %v\n", chs[0], err)
		}
		stop, err := strconv.Atoi(chs[1])
		if err != nil {
			log.Fatalf("Could not convert %v to a valid chapter: %v\n", chs[1], err)
		}
		if start < stop {
			for i := start; i <= stop; i++ {
				if _, exists := (*entryMap)[i]; !exists {
					*ran = append(*ran, i)
					(*entryMap)[i] = struct{}{}
				}
			}
		} else {
			for i := stop; i <= start; i++ {
				if _, exists := (*entryMap)[i]; !exists {
					*ran = append(*ran, i)
					(*entryMap)[i] = struct{}{}
				}
			}
		}
	} else if len(chs) == 1 {
		start, err := strconv.Atoi(chs[0])
		log.Fatalf("Could not convert %v to a valid chapter: %v\n", chs[0], err)
		if _, exists := (*entryMap)[start]; !exists {
			*ran = append(*ran, start)
			(*entryMap)[start] = struct{}{}
		}
	}
}

// convert command line chapter args to their int equivalents
func GetChapterRangeFromArgs(vals *[]string) *[]int {
	chapters := []int{}
	chapterMap := make(map[int]struct{})
	for _, val := range *vals {
		if strings.Contains(val, "-") {
			GetChapterRange(val, &chapters, &chapterMap)
		} else {
			x, err := strconv.Atoi(val)
			if err != nil {
				log.Fatalf("%v could not be converted to a valid chapter: %v\n", val, err)
			}
			if _, exists := chapterMap[x]; !exists {
				chapters = append(chapters, x)
				chapterMap[x] = struct{}{}
			}
		}
	}
	fmt.Printf("chapters: %v\n", chapters)
	return &chapters
}

// just something to help with tests
type ReadWrite struct {
	ReadFrom io.Reader
	WriteTo  io.Writer
}

func GetMatchFromSearchResults(r ReadWrite, results []scraper.Manga) scraper.Manga {
	fmt.Fprint(r.WriteTo, "Id \t Manga\n")
	ids := make(map[int]scraper.Manga)
	for i, m := range results {
		fmt.Fprintf(r.WriteTo, "%d \t %s\n", i+1, m.MangaName)
		ids[i+1] = m
	}

	myScanner := bufio.NewScanner(r.ReadFrom)
	fmt.Fprintf(r.WriteTo, "Enter the id of the correct manga: ")
	var (
		id  int
		err error
	)
	for myScanner.Scan() {
		id, err = strconv.Atoi(myScanner.Text())
		if err != nil {
			log.Printf("Try again. err %v\n", err)
		} else {
			if _, ok := ids[id]; !ok {
				fmt.Fprintf(r.WriteTo, "Enter a valid Id, please: ")
			} else {
				break
			}
		}
	}

	match := ids[id]
	return match
}

func cbzify(folderPath string) error {
	cbzFile, err := os.Create(folderPath + ".cbz")
	if err != nil {
		return err
	}

	zipWriter := zip.NewWriter(cbzFile)
	err = filepath.Walk(folderPath,
		func(filePath string, fileInfo os.FileInfo, err error) error {
			if err != nil || fileInfo.IsDir() {
				return err
			}

			relativeFilePath, err := filepath.Rel(folderPath, filePath)
			if err != nil {
				return err
			}
			archivePath := path.Join(filepath.SplitList(relativeFilePath)...)

			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			zipFileWriter, err := zipWriter.Create(archivePath)
			if err != nil {
				return err
			}

			_, err = io.Copy(zipFileWriter, file)
			if err != nil {
				return err
			}

			return nil
		})
	if err != nil {
		return err
	}

	err = zipWriter.Close()
	if err != nil {
		return err
	}

	err = os.RemoveAll(folderPath)
	if err != nil {
		fmt.Printf("Couldn't delete %v after creating cbz\n", folderPath)
		return nil
	}

	return nil
}

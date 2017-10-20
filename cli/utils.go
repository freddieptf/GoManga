package cli

import (
	"archive/zip"
	"bufio"
	"fmt"
	scraper "github.com/freddieptf/manga-scraper/scraper"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type imgItem struct {
	URL string
	ID  int
}

type searchResult struct {
	manga, mangaID string
}

func getRange(vals *[]string) *[]int {
	chapters := []int{}
	for _, val := range *vals {
		if strings.Contains(val, "-") {
			chs := strings.Split(val, "-")
			chInts := []int{}
			for _, chapter := range chs {
				x, err := strconv.Atoi(chapter)
				if err != nil {
					log.Printf("%v could not be converted to a chapter.\n", chapter)
					log.Fatal(err)
				}
				chInts = append(chInts, x)
			}
			sort.Ints(chInts)
			for i := chInts[0]; i <= chInts[len(chInts)-1]; i++ {
				chapters = append(chapters, i)
			}
		} else {
			x, err := strconv.Atoi(val)
			if err != nil {
				log.Printf("%v could not be converted to a chapter.\n", val)
				log.Fatal(err)
			}
			chapters = append(chapters, x)
		}
	}
	fmt.Printf("chapters: %v\n", chapters)
	return &chapters
}

func getMatchFromSearchResults(results []scraper.Manga) scraper.Manga {
	fmt.Printf("Id \t Manga\n")
	for i, m := range results {
		fmt.Printf("%d \t %s\n", i, m.MangaName)
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
	if id > len(results) {
		fmt.Printf("Insert one of the Ids in the results, please: ")
		goto scanDem
	}
	match := results[id]
	return match
}

func (item *imgItem) downloadImage(path string) error {
	response, err := http.Get(item.URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	imgPath := filepath.Join(path, strconv.Itoa(item.ID)+".jpg")
	err = ioutil.WriteFile(imgPath, body, 0655)
	if err != nil {
		return err
	}
	return nil
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

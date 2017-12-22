package main

// import (
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"os"
// 	"path/filepath"
// 	"sort"
// 	"strconv"
// 	"strings"
// )

// type byChapter []os.FileInfo

// //implements sort.Interface based on
// // the chapter int field field.
// func (a byChapter) Len() int      { return len(a) }
// func (a byChapter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
// func (a byChapter) Less(i, j int) bool {
// 	m := strings.Split(a[i].Name(), ": ")
// 	n := strings.Split(a[j].Name(), ": ")
// 	mi, _ := strconv.ParseFloat(m[0], 32)
// 	ni, _ := strconv.ParseFloat(n[0], 32)
// 	return mi < ni
// }

// func updateMangaLib() {
// 	parentPath := filepath.Join(os.Getenv("HOME"), "Manga")
// 	_, err := os.Open(parentPath)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	updateReaderLib(parentPath)
// 	updateFoxLib(parentPath)

// }

// func updateReaderLib(parentPath string) {
// 	mangaReaderPath := filepath.Join(parentPath, "MangaReader")
// 	var currentReaderChapters []readerChapter

// 	if f, err := os.Open(mangaReaderPath); err == nil {
// 		mangaFolders, err := f.Readdirnames(0)
// 		if err != nil {
// 			log.Println(err)
// 			return
// 		}

// 		for _, folderpath := range mangaFolders {
// 			chapters, err := ioutil.ReadDir(filepath.Join(mangaReaderPath, folderpath))
// 			if err != nil {
// 				log.Println(err)
// 			}

// 			sort.Sort(byChapter(chapters))
// 			currentReaderChapters = append(currentReaderChapters,
// 				readerChapter{
// 					manga:   folderpath,
// 					chapter: strings.Split(chapters[len(chapters)-1].Name(), ": ")[0],
// 				})

// 		}

// 		fmt.Printf("reader: %v\n", currentReaderChapters)
// 	}

// }

// func updateFoxLib(parentPath string) {
// 	mangaFoxPath := filepath.Join(parentPath, "MangaFox")
// 	var currentFoxChapters []foxChapter

// 	if f, err := os.Open(mangaFoxPath); err == nil {
// 		mangaFolders, err := f.Readdirnames(0)
// 		if err != nil {
// 			log.Println(err)
// 		}

// 		for _, folderpath := range mangaFolders {
// 			chapters, err := ioutil.ReadDir(filepath.Join(mangaFoxPath, folderpath))
// 			if err != nil {
// 				log.Println(err)
// 			}

// 			sort.Sort(byChapter(chapters))
// 			currentFoxChapters = append(currentFoxChapters,
// 				foxChapter{
// 					manga:   folderpath,
// 					chapter: strings.Split(chapters[len(chapters)-1].Name(), ": ")[0],
// 				})

// 		}

// 		fmt.Printf("fox: %v\n", currentFoxChapters)

// 	}
// }

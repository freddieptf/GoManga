package scraper

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// return the results and the search query
func getTestSearchResults(t *testing.T) ([]Manga, string) {
	testManga := "kengan"
	foxSource := &FoxManga{}
	results, err := foxSource.Search(testManga)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) <= 0 {
		t.Fatal("search results of length 0")
	}
	return results, testManga
}

func TestFoxGetMangaDetails(t *testing.T) {
	results, _ := getTestSearchResults(t)
	manga := results[0]
	doc, err := makeDocRequest(fmt.Sprintf("%s/%s", foxURL, manga.MangaID))
	if err != nil {
		t.Fatal(err)
	}
	details := getMangaDetails(doc)
	if details.name != manga.MangaName {
		t.Error("manga details don't match")
	}
}

func testGetChapterUrlFromListing(t *testing.T) (chapterURL string) {
	results, _ := getTestSearchResults(t)
	manga := results[0]
	doc, err := makeDocRequest(fmt.Sprintf("%s/%s", foxURL, manga.MangaID))
	if err != nil {
		t.Fatal(err)
	}
	chapterURL, chapterTitle := getChapterUrlFromListing("7.5", doc)
	if _, err := url.ParseRequestURI(chapterURL); err != nil {
		t.Fatalf("chaterURL=%s : %v\n", chapterURL, err)
	}
	if chapterTitle == "" {
		t.Error("returned empty chapter title")
	}
	return
}

func testGetFoxChPageUrls(t *testing.T) (chapterPageUrls []string) {
	chapterURL := testGetChapterUrlFromListing(t)
	doc, err := makeDocRequest(chapterURL)
	if err != nil {
		t.Fatal(err)
	}
	chapterPageUrls = getFoxChPageUrls(doc)
	if len(chapterPageUrls) <= 0 {
		t.Fatal("couldn't get the chapter page urls")
	}
	for _, chapterPageURL := range chapterPageUrls {
		if _, err := url.ParseRequestURI(chapterPageURL); err != nil {
			t.Errorf("chapterPageURL=%s : %v\n", chapterPageURL, err)
		}
	}
	return
}

func TestGetFoxChPageImgUrl(t *testing.T) {
	chapterPageUrls := testGetFoxChPageUrls(t)
	samplePageUrl := chapterPageUrls[0]
	imgUrl := getFoxChPageImgUrl(samplePageUrl)
	if _, err := url.ParseRequestURI(imgUrl); err != nil {
		t.Error(err)
	}
}

func TestFoxSearch(t *testing.T) {
	results, query := getTestSearchResults(t)
	for _, manga := range results {
		if !strings.Contains(strings.ToLower(manga.MangaName), query) {
			t.Error("results could be wrong")
		}
	}
}

func TestFoxGetChapter(t *testing.T) {
	results, _ := getTestSearchResults(t)
	manga := results[0]
	foxSource := &FoxManga{}
	chapter, err := foxSource.GetChapter(manga.MangaID, "1")
	if err != nil {
		t.Error(err)
	}
	if len(chapter.ChapterPages) <= 0 {
		t.Error("no chapter pages returned")
	}
}

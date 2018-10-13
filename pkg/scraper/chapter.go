package scraper

type ChapterPage struct {
	Page int
	Url  string
}

type Chapter struct {
	MangaName    string
	ChapterTitle string
	VolumeTitle  string
	SourceName   string
	ChapterPages []ChapterPage
}

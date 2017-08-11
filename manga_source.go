package main

type mangaSource interface {
	search() (map[int]searchResult, error)
	getChapters(n int)
	getVolumes(n int)
}

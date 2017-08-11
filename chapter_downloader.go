package main

import (
	"fmt"
	"gopkg.in/go-playground/pool.v1"
)

type chapterSource interface {
	getChapter() error
	getMangaName() string
	getChapterName() string
}

type chapterDownloader struct {
	active    int
	totalJobs int
	pool      *pool.Pool
	jobFn     pool.JobFunc
}

func (dwn chapterDownloader) init(n, c int) *chapterDownloader {
	downloadPool := pool.NewPool(n, c)
	jobFn := func(job *pool.Job) {
		chSrc := job.Params()[0].(chapterSource)
		e := chSrc.getChapter()
		if e != nil {
			fmt.Printf("Download Failed: %v chapter %v (%v)\n", chSrc.getMangaName(), chSrc.getChapterName(), e)
			return
		}
		job.Return(chSrc.getMangaName() + " chapter " + chSrc.getChapterName())
	}
	return &chapterDownloader{
		active:    n,
		totalJobs: c,
		pool:      downloadPool,
		jobFn:     jobFn,
	}
}

func (dwn *chapterDownloader) queue(chSrc chapterSource) {
	dwn.pool.Queue(dwn.jobFn, chSrc)
}

func (dwn *chapterDownloader) startDownloads() <-chan interface{} {
	return dwn.pool.Results()
}

package cli

import (
	"io"
	"io/ioutil"
	"reflect"
	"testing"

	scraper "github.com/freddieptf/manga-scraper/pkg/scraper"
)

func TestGetChapterRangeFromArg(t *testing.T) {
	for _, arg := range []struct {
		val      string
		ran      *[]int
		entryMap *map[int]struct{}
		expected *[]int
	}{
		{"1-3", &[]int{}, &map[int]struct{}{}, &[]int{1, 2, 3}},
		{"1-3", &[]int{}, &map[int]struct{}{1: {}}, &[]int{2, 3}},
		{"1-1", &[]int{}, &map[int]struct{}{}, &[]int{1}},
		{"1-1", &[]int{}, &map[int]struct{}{1: {}}, &[]int{}},
		{"-3", &[]int{}, &map[int]struct{}{1: {}}, &[]int{3}},
	} {
		getChapterRange(arg.val, arg.ran, arg.entryMap)
		if !reflect.DeepEqual(arg.ran, arg.expected) {
			t.Errorf("GetChapterRange(%v) = %v, want %v", arg.val, arg.ran, arg.expected)
		}
	}
}

func TestGetChapterRangeFromArgs(t *testing.T) {
	for _, arg := range []struct {
		input  *[]string
		output *[]int
	}{
		{&[]string{"1", "3-4"}, &[]int{1, 3, 4}},
		{&[]string{"1-1", "3-4", "3-4", "3", "4"}, &[]int{1, 3, 4}},
		{&[]string{"1-4", "3-4", "3", "4"}, &[]int{1, 2, 3, 4}},
		{&[]string{"3-1", "3-", "3-4", "-4"}, &[]int{1, 2, 3, 4}},
		{&[]string{"3-1", "-", "2-1", "3-4"}, &[]int{1, 2, 3, 4}},
		{&[]string{"1-4", "7", "9-10"}, &[]int{1, 2, 3, 4, 7, 9, 10}},
	} {
		o := getChapterRangeFromArgs(arg.input)
		if !reflect.DeepEqual(o, arg.output) {
			t.Errorf("GetChapterRangeFromArgs(%v) = %v, want %v", arg.input, o, arg.output)
		}
	}
}

type userInput string

func (in *userInput) Read(p []byte) (n int, err error) {
	n = copy(p, *in)
	*in = (*in)[n:]
	if n == 0 {
		err = io.EOF
	}
	return
}

func TestGetMatchFromSearchResults(t *testing.T) {
	manga := []struct {
		mangas   []scraper.Manga
		input    string
		expected scraper.Manga
	}{
		{[]scraper.Manga{{"test", "/test/url"}, {"test1", "/test/url/1"}}, "2", scraper.Manga{"test1", "/test/url/1"}},
		{[]scraper.Manga{{"test", "/test/url"}, {"test1", "/test/url/1"}}, "1", scraper.Manga{"test", "/test/url"}},
	}

	for _, arg := range manga {
		r := userInput(arg.input)
		m := getMatchFromSearchResults(readWrite{&r, ioutil.Discard}, arg.mangas)
		if !reflect.DeepEqual(m, arg.expected) {
			t.Errorf("getMatchFromSearchResults(%v) = %v, want %v", arg.input, m, arg.expected)
		}
	}

}

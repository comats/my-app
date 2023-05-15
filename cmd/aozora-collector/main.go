package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"path"
	"net/url"
	"net/http"
	"io/ioutil"
	"archive/zip"
	"bytes"
	
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/japanese"
)

type Entry struct {
	AuthorID string
	Author string
	TitleID string
	Title string
	SiteURL string
	ZipURL string
}

func findEntries(siteURL string) ([]Entry, error) {
	doc, err := goquery.NewDocument(siteURL)
	if err != nil {
		return nil, err
	}
	pat := regexp.MustCompile(`.*/cards/([0-9]+)/card([0-9]+).html$`)
	entries := []Entry{}
	doc.Find("li a").Each(func(n int, elem *goquery.Selection) {
		token := pat.FindStringSubmatch(elem.AttrOr("href", ""))
		if len(token) !=3 {
			return
		}
		title := elem.Text()
		siteURL := fmt.Sprintf("https://www.aozora.gr.jp/cards/%s/card%s.html", token[1], token[2])
		author, zipURL := findAuthorAndZip(siteURL)
		if zipURL != "" {
			entries = append(entries, Entry{
				AuthorID: token[1],
				Author: author,
				TitleID: token[2],
				Title: title,
				SiteURL: siteURL,
				ZipURL: zipURL,
			})
		}
	})
	return entries, nil
}

func findAuthorAndZip(siteURL string) (string, string) {
	doc, err := goquery.NewDocument(siteURL)
	if err != nil {
		return "", ""
	}

	author := doc.Find("table[summary=作家データ] tr:nth-child(1) td:nth-child(2)").Text()

	zipURL := ""
	doc.Find("table.download a").Each(func(n int, elem *goquery.Selection) {
		href := elem.AttrOr("href", "")
		if strings.HasSuffix(href, ".zip") {
			zipURL = href
		}
	})
	u, err := url.Parse(siteURL)
	if err != nil {
		return author, ""
	}
	u.Path = path.Join(path.Dir(u.Path), zipURL)
	return author, u.String()
}

func extractText(zipURL string) (string, error) {
	resp, err := http.Get(zipURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	r, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return "", err
	}
	for _, file := range r.File {
		if path.Ext(file.Name) == ".txt" {
			f, err := file.Open()
			if err != nil {
				return "", err
			}
			b, err := ioutil.ReadAll(f)
			f.Close()
			if err != nil {
				return "", err
			}
			b, err = japanese.ShiftJIS.NewDecoder().Bytes(b)
			return string(b), nil
		}
	}
	return "", nil
}

func main() {
	listURL := "https://www.aozora.gr.jp/index_pages/person879.html"

	entries, err := findEntries(listURL)
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {
		content, err := extractText(entry.ZipURL)
		println(content)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

package scraper

import (
	"strings"

	"github.com/gocolly/colly"
)

type Patch struct {
	Date        string
	Version     string
	Description string
	Link        string
}

func GetPatchNotes() Patch {
	base_url := "https://playvalorant.com/en-us/news/game-updates/"

	c := colly.NewCollector()

	found := false

	var patch Patch

	c.OnHTML(`[data-testid="card-body"]`, func(e *colly.HTMLElement) {
		if found {
			return
		}

		date := e.ChildText("time")
		version := e.ChildText(`[data-testid="card-title"]`)
		description := e.ChildText(`[data-testid="card-description"]`)
		link := "https://playvalorant.com" + e.DOM.ParentsFiltered("a").AttrOr("href", "")

		patch = Patch{Date: date, Version: version, Description: description, Link: link}

		if strings.Contains(version, "VALORANT Patch Notes") {
			found = true
		}
	})

	c.Visit(base_url)

	return patch
}

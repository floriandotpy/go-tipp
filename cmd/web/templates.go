package main

import (
	"html/template"
	"path/filepath"

	"tipp.casualcoding.com/internal/models"
)

type templateData struct {
	MatchTipps []models.MatchTipp
	Matches    []models.Match
	T          map[string]string
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := filepath.Glob("./ui/html/pages/*html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		files := []string{
			"./ui/html/base.html",
			"./ui/html/partials/nav.html",
			page,
		}
		ts, err := template.ParseFiles(files...)
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}

	return cache, nil
}

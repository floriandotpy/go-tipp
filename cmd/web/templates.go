package main

import (
	"tipp.casualcoding.com/internal/models"
)

type templateData struct {
	MatchTipps []models.MatchTipp
	Matches    []models.Match
	T          map[string]string
}

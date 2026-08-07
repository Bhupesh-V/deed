package models

type Input struct {
	DSN    string
	Tables []string
	Count  int64
	Config string
}

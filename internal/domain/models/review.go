package models

type Review struct {
	ID      uint
	DraftID uint
	Author  string
	Notes   string
}

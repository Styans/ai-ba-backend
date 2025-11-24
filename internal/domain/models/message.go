package models

type Message struct {
	ID        uint
	SessionID uint
	Author    string // "user" или "ai"
	Text      string
}

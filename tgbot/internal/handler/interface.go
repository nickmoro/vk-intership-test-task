package handler

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type BotHandler interface {
	// HandleUpdates parses update and run command handler func as new goroutine.
	// You can have many HandleUpdates running simultaneously.
	HandleUpdates(updates <-chan tgbotapi.Update)

	Set(msg *tgbotapi.Message) (string, error)
	Get(msg *tgbotapi.Message) (string, error)
	Del(msg *tgbotapi.Message) (string, error)
}

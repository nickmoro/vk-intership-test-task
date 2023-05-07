package handler

import (
	"fmt"
	"strings"
	"tgbot/internal/repo"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	internalErrorMessage = "Произошла внутренняя ошибка. Пожалуйста, попробуйте позже"

	helpMessage = "Доступные команды:\n" +
		"/set Имя сервиса Логин Пароль — сохранить логин-пароль для сервиса" +
		"(предыдущие данные для этого сервиса будут удалены)\n" +
		"/get Имя сервиса — получить логин-пароль к сервису\n" +
		"/del Имя сервиса — отвязать логин-пароль от сервиса\n" +
		"Пример: `/set Мой сервис my_login my_password`\n" +
		"Примечание: Имя сервиса может содержать пробелы, Логин и Пароль — нет"

	// AvailableLetters can be used in servicename, login and password.
	availableLetters = " abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"абвгдеёжзийклмнопрстуфхцчшщъыьэюяАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ" +
		"!#$%&()*+,-./0123456789:;<=>?@[]^_{|}~;"

	messageDeleteDelay = 60 * time.Second
)

type Handler struct {
	logger          *zap.SugaredLogger
	bot             *tgbotapi.BotAPI
	repo            repo.NotesRepo
	commandHandlers map[string]func(msg *tgbotapi.Message) (string, error)
}

func NewBotHandler(logger *zap.SugaredLogger, bot *tgbotapi.BotAPI,
	repo repo.NotesRepo) BotHandler {

	h := &Handler{logger: logger, bot: bot, repo: repo}
	h.commandHandlers = map[string]func(msg *tgbotapi.Message) (string, error){
		"set": h.Set,
		"get": h.Get,
		"del": h.Del,
	}
	return h
}

// HandleUpdates parses update and run command handler func as new goroutine.
// You can have many HandleUpdates running simultaneously.
func (h *Handler) HandleUpdates(updates <-chan tgbotapi.Update) {
	for update := range updates {
		msg := update.Message
		if msg == nil || msg.Chat == nil {
			continue
		}

		reply := tgbotapi.NewMessage(msg.Chat.ID, "")
		reply.ReplyToMessageID = msg.MessageID

		handlerFunc, found := h.commandHandlers[msg.Command()]

		if !found {
			// Invalid command (commandHandler not found)
			reply.Text = helpMessage

			err := h.send(reply)
			if err != nil {
				h.logger.Error(errors.Wrap(err, "h.send"))
			}

			continue
		}

		symbol, found := findSymbolFromSetInString(msg.Text[1:], availableLetters)

		if found {
			// Valid command with invalid symbol(s) in arguments
			reply.Text = fmt.Sprintf(
				"Невозможно обработать запрос, сообщение содержит недопустимый символ: %v", symbol)

			err := h.send(reply)
			if err != nil {
				h.logger.Error(errors.Wrap(err, "h.send"))
			}
			continue
		}

		h.logger.Debugf(`chat %v: Requested "%v"`, msg.Chat.ID, msg.Command())

		// run command handler
		go func(msg *tgbotapi.Message) {
			start := time.Now()
			text, err := handlerFunc(msg)

			if err == nil {
				h.logger.Debugf(`chat %v: Command "%v" served in %v ms`,
					msg.Chat.ID, msg.Command(), time.Since(start).Milliseconds())
			} else {
				text = internalErrorMessage
				h.logger.Errorf(`chat %v: Error serving command "%v": %v`,
					msg.Chat.ID, msg.Command(), err)
			}

			reply.Text = text
			reply.ParseMode = "MarkDown"

			err = h.sendConfident(reply)
			if err != nil {
				h.logger.Error(errors.Wrap(err, "h.sendConfident"))
			}
		}(msg)
	}
}

// send sends msg and deletes user's message after messageDeleteDelay.
func (h *Handler) send(msg tgbotapi.MessageConfig) error {
	_, err := h.bot.Send(msg)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf(`h.bot.Send "%v"`, msg))
	}

	time.Sleep(messageDeleteDelay)

	delMsg := tgbotapi.NewDeleteMessage(msg.ChatID, msg.ReplyToMessageID)

	_, err = h.bot.Send(delMsg)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf(`h.bot.Send "%v"`, delMsg))
	}

	return nil
}

// sendConfident sends msg and deletes both messages after messageDeleteDelay.
func (h *Handler) sendConfident(msg tgbotapi.MessageConfig) error {

	// Confident messages not placed into logs
	// We believe that handlerFuncs provided valid msg

	msgToUser, err := h.bot.Send(msg)
	if err != nil {
		return errors.Wrap(err, "h.bot.Send")
	}

	time.Sleep(messageDeleteDelay)

	delMsg := tgbotapi.NewDeleteMessage(msg.ChatID, msg.ReplyToMessageID)
	_, err = h.bot.Send(delMsg)
	if err != nil {
		return errors.Wrap(err, "h.bot.Send")
	}

	_, err = h.bot.Send(tgbotapi.NewDeleteMessage(msg.ChatID, msgToUser.MessageID))
	if err != nil {
		return errors.Wrap(err, "h.bot.Send")
	}

	return nil
}

func findSymbolFromSetInString(str, set string) (string, bool) {
	for _, symbol := range str {
		if !strings.Contains(availableLetters, string(symbol)) {
			return string(symbol), true
		}
	}
	return "", false
}

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
	InternalErrorMessage = "Произошла внутренняя ошибка. Пожалуйста, попробуйте позже"

	HelpMessage = "Доступные команды:\n" +
		"/set Имя сервиса Логин Пароль — сохранить логин-пароль для сервиса" +
		"(предыдущие данные для этого сервиса будут удалены)\n" +
		"/get Имя сервиса — получить логин-пароль к сервису\n" +
		"/del Имя сервиса — отвязать логин-пароль от сервиса\n" +
		"Пример: `/set Мой сервис my_login my_password`\n" +
		"Примечание: Имя сервиса может содержать пробелы, Логин и Пароль — нет"
)

type Handler struct {
	Logger          *zap.SugaredLogger
	Bot             *tgbotapi.BotAPI
	Repo            repo.NotesRepo
	CommandHandlers map[string]func(msg *tgbotapi.Message) (string, error)
}

func NewBotHandler(logger *zap.SugaredLogger, bot *tgbotapi.BotAPI,
	repo repo.NotesRepo) BotHandler {

	h := &Handler{
		Logger: logger,
		Bot:    bot,
		Repo:   repo,
	}

	h.CommandHandlers = map[string]func(msg *tgbotapi.Message) (string, error){
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

		h.Logger.Debugf(`chat %v: Requested "%v"`, msg.Chat.ID, msg.Command())

		handlerFunc, found := h.CommandHandlers[msg.Command()]
		if !found {
			reply := tgbotapi.NewMessage(msg.Chat.ID, HelpMessage)
			reply.ReplyToMessageID = msg.MessageID

			_, err := h.Bot.Send(reply)
			if err != nil {
				h.Logger.Error(errors.Wrap(err, "h.Bot.Send"))
			}

			continue
		}

		// run command handler
		go func(msg *tgbotapi.Message) {
			start := time.Now()

			text, err := handlerFunc(msg)

			if err == nil {
				h.Logger.Debugf(`chat %v: Command "%v" served in %v ms`,
					msg.Chat.ID, msg.Command(), time.Since(start).Milliseconds())
			} else {
				text = InternalErrorMessage
				h.Logger.Errorf(`chat %v: Error serving command "%v": %v`,
					msg.Chat.ID, msg.Command(), err)
			}

			reply := tgbotapi.NewMessage(msg.Chat.ID, text)
			reply.ReplyToMessageID = msg.MessageID
			reply.ParseMode = "MarkDown"

			_, err = h.Bot.Send(reply)
			if err != nil {
				h.Logger.Error(errors.Wrap(err, "h.Bot.Send"))
			}
		}(msg)
	}
}

// Set is multithreading-friendly "/set" command handler.
func (h *Handler) Set(msg *tgbotapi.Message) (string, error) {
	input := strings.Split(msg.Text, " ")
	if len(input) < 4 {
		text := "Некорректный ввод.\n" +
			"/set Имя сервиса Логин Пароль — сохранить логин-пароль для сервиса" +
			"(предыдущие данные для этого сервиса будут удалены)\n" +
			"Пример: `/set Мой сервис my_login my_password`"
		return text, nil
	}

	serviceName := input[1]

	// handle multi-words service name
	for i := 2; i < len(input)-2; i++ {
		serviceName += " " + input[i]
	}

	login := input[len(input)-2]
	password := input[len(input)-1]

	note := repo.Note{
		ServiceName: serviceName,
		Login:       login,
		Password:    password,
	}

	err := h.Repo.Set(fmt.Sprint(msg.Chat.ID), note)
	if err != nil {
		return "", errors.Wrap(err, "h.Repo.Set")
	}

	text := fmt.Sprintf("Сервис: `%v`\n"+
		"Логин: `%v`\n"+
		"Пароль: `%v`",
		note.ServiceName, note.Login, note.Password)
	return text, nil
}

// Get is multithreading-friendly "/get" command handler.
func (h *Handler) Get(msg *tgbotapi.Message) (string, error) {
	input := strings.Split(msg.Text, " ")
	if len(input) < 2 {
		text := "Некорректный ввод.\n" +
			"/get Имя сервиса — получить логин-пароль к сервису\n" +
			"Например, `/get Мой сервис`"
		return text, nil
	}

	serviceName := input[1]

	// handle multi-words service name
	for i := 2; i < len(input); i++ {
		serviceName += " " + input[i]
	}

	note, err := h.Repo.Get(fmt.Sprint(msg.Chat.ID), serviceName)

	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			return "", errors.Wrap(err, "h.Repo.Get")
		}
		text := fmt.Sprintf("Сервис с именем `%v` не найден", serviceName)
		return text, nil
	}

	text := fmt.Sprintf("Сервис: `%v`\n"+
		"Логин: `%v`\n"+
		"Пароль: `%v`",
		note.ServiceName, note.Login, note.Password,
	)

	return text, nil
}

// Del is multithreading-friendly "/del" command handler.
func (h *Handler) Del(msg *tgbotapi.Message) (string, error) {
	input := strings.Split(msg.Text, " ")
	if len(input) < 2 {
		text := "Некорректный ввод.\n" +
			"/del Имя сервиса — отвязать логин-пароль от сервиса\n" +
			"Например, `/del Мой сервис`"
		return text, nil
	}

	serviceName := input[1]

	// handle multi-words service name
	for i := 2; i < len(input); i++ {
		serviceName += " " + input[i]
	}

	err := h.Repo.Del(fmt.Sprint(msg.Chat.ID), serviceName)
	text := fmt.Sprintf("Логин-пароль отвязан от сервиса `%v`", serviceName)

	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			return "", errors.Wrap(err, "h.Repo.Del")
		}
		text = fmt.Sprintf("Логин-пароль к сервису `%v` не найден", serviceName)
	}

	return text, nil
}

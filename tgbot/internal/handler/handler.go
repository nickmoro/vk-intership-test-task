package handler

import (
	"fmt"
	"strings"
	"tgbot/internal/repo"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Handler struct {
	Logger          *zap.SugaredLogger
	Bot             *tgbotapi.BotAPI
	Repo            repo.NotesRepo
	CommandHandlers map[string]func(msg *tgbotapi.Message) (tgbotapi.MessageConfig, error)
}

func NewBotHandler(logger *zap.SugaredLogger, bot *tgbotapi.BotAPI,
	repo repo.NotesRepo) BotHandler {

	h := &Handler{
		Logger: logger,
		Bot:    bot,
		Repo:   repo,
	}

	h.CommandHandlers = map[string]func(msg *tgbotapi.Message) (tgbotapi.MessageConfig, error){
		"set": h.Set,
		"get": h.Get,
		"del": h.Del,
	}

	return h
}

const (
	InternalErrorMessage = "Произошла внутренняя ошибка. Пожалуйста, попробуйте позже"
)

func (h *Handler) WorkerFunc(updates <-chan tgbotapi.Update) {
	for update := range updates {
		msg := update.Message
		if msg == nil || msg.Chat == nil {
			continue
		}
		h.Logger.Debugf("Message from %v", msg.Chat.ID)

		reply, err := getHelp(msg)
		handlerFunc, found := h.CommandHandlers[msg.Command()]
		if found {
			reply, err = handlerFunc(msg)
		}
		// "else" is not required, variables are already declared with default values

		if err != nil {
			h.Logger.Errorf(`Error serving command "%v": %v`, msg.Command(), err)
			reply = tgbotapi.NewMessage(msg.Chat.ID, InternalErrorMessage)
		}

		reply.ReplyToMessageID = msg.MessageID

		_, err = h.Bot.Send(reply)
		if err != nil {
			h.Logger.Error(err)
		}
	}
}

func getHelp(msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	text := "Доступные команды:\n" +
		"/set SERVICE LOGIN PASSWORD -- сохранить логин-пароль для сервиса\n" +
		"/get SERVICE -- получить логин-пароль к сервису\n" +
		"/del SERVICE -- отвязать логин-пароль от сервиса\n" +
		"Замените SERVICE, LOGIN и PASSWORD на название сервиса (одним словом), логин и пароль соответственно"
	return tgbotapi.NewMessage(msg.Chat.ID, text), nil
}

func (h *Handler) Set(msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	input := strings.Split(msg.Text, " ")
	if len(input) != 4 {
		text := "Некорректный ввод.\n" +
			"/set SERVICE LOGIN PASSWORD -- сохранить логин-пароль для сервиса\n" +
			"Например, /set pentagon admin qwerty12345"
		return tgbotapi.NewMessage(msg.Chat.ID, text), nil
	}

	note := repo.Note{
		ServiceName: input[1],
		Login:       input[2],
		Password:    input[3],
	}

	err := h.Repo.Set(fmt.Sprint(msg.Chat.ID), note)
	if err != nil {
		return tgbotapi.MessageConfig{}, errors.Wrap(err, "h.Repo.Set")
	}

	text := fmt.Sprintf(`Сервис "%v":\nЛогин: %v\nПароль: %v`,
		note.ServiceName, note.Login, note.Password)
	return tgbotapi.NewMessage(msg.Chat.ID, text), nil
}

func (h *Handler) Get(msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	input := strings.Split(msg.Text, " ")
	if len(input) != 2 {
		text := "Некорректный ввод.\n" +
			"/get SERVICE -- получить логин-пароль к сервису\n" +
			"Например, /get pentagon"
		return tgbotapi.NewMessage(msg.Chat.ID, text), nil
	}

	note, err := h.Repo.Get(fmt.Sprint(msg.Chat.ID), input[1])
	if err != nil {
		return tgbotapi.MessageConfig{}, errors.Wrap(err, "h.Repo.Get")
	}

	text := fmt.Sprintf(`Сервис "%v":\nЛогин: %v\nПароль: %v`,
		note.ServiceName, note.Login, note.Password)
	return tgbotapi.NewMessage(msg.Chat.ID, text), nil
}

func (h *Handler) Del(msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	input := strings.Split(msg.Text, " ")
	if len(input) != 2 {
		text := "Некорректный ввод.\n" +
			"/del SERVICE -- отвязать логин-пароль от сервиса\n" +
			"Например, /del pentagon"
		return tgbotapi.NewMessage(msg.Chat.ID, text), nil
	}

	err := h.Repo.Del(fmt.Sprint(msg.Chat.ID), input[1])
	if err != nil {
		return tgbotapi.MessageConfig{}, errors.Wrap(err, "h.Repo.Get")
	}

	text := fmt.Sprintf(`Логин-пароль отвязан от сервиса "%v"`, input[1])
	return tgbotapi.NewMessage(msg.Chat.ID, text), nil
}

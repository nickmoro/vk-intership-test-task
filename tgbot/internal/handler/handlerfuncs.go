package handler

import (
	"fmt"
	"strings"
	"tgbot/internal/repo"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
)

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

	err := h.repo.Set(fmt.Sprint(msg.Chat.ID), note)
	if err != nil {
		return "", errors.Wrap(err, "h.repo.Set")
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

	note, err := h.repo.Get(fmt.Sprint(msg.Chat.ID), serviceName)

	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			return "", errors.Wrap(err, "h.repo.Get")
		}
		text := fmt.Sprintf("Логин-пароль к сервису `%v` не найден", serviceName)
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

	err := h.repo.Del(fmt.Sprint(msg.Chat.ID), serviceName)
	text := fmt.Sprintf("Логин-пароль отвязан от сервиса `%v`", serviceName)

	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			return "", errors.Wrap(err, "h.repo.Del")
		}
		text = fmt.Sprintf("Логин-пароль к сервису `%v` не найден", serviceName)
	}

	return text, nil
}

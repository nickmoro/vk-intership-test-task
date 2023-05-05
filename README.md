# Телеграм бот для хранения паролей
Выполнил Гапонов Н. в качестве тестового задания к вакансии https://internship.vk.company/vacancy/626

## Описание задачи
Реализовать Telegram бота, обладающего функционалом персонального хранилища паролей.  
Поддерживаются следующие команды:
1. /set -- добавить логин и пароль к сервису
2. /get -- получить логин и пароль по названию сервиса
3. /del -- удалить значения для сервиса

Выполнены основные требования к реализации:
1. Бот написан на Golang
2. Сообщения с паролями удаляются по истечении заданного константой времени
3. Каждому пользователю выводятся исключительно его пароли (для этого учитывается ID чата telegram)

Также выполнены дополнительные требования к реализации:
1. ~Надеюсь, что~ контейнер запущен на 87.239.111.2, а сам бот доступен по ссылке http://t.me/password_keeper_by_booec_bot
2. Можно запустить бота при помощи `docker compose up`
3. Для хранения данных используется mongoDB.

## Описание решения
```
├── cmd  
│   └── tgbot  
│       ├── main.go: Инициализация логгера (zap.SugaredLogger) и бота, запуск воркеров.  
│       └── secret.go: Файл не импортирован в git, здесь хранится ключ бота.  
└── internal  
    └── handler: Пакет, реализующий обработчик команд пользователя. Для работы необходим NotesRepo.  
        ├── handler.go: Реализация хендлера.  
        └── interface.go: Описание интерфейса хендлера.  
```
**Примечание:** Для локального запуска задайте `BotToken` в main.go. В целях безопасности он не импортирован в git-репозиторий.


*Далее идёт вольное описание кода*  
При реализации хендлера, я решил продумать возможность обрабатывать запросы многопоточно. Заметим, что
`tgbotapi.UpdatesChannel` ꟷ просто канал апдейтов, в связи с чем можно применить Worker Pool: создать несколько
горутин (их количество регулируется константой), которые извлекают данные из канала и обрабатывают запросы независимо.
Очевидно, что конкретное значение из канала сможет считать только одна горутина, даже если свободных горутин несколько.

Далее я описал необходимые мне структуры, интерфейс хендлера и приступил к его реализации (воркером является HandleFunc):
```golang
package handler

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

type User struct {
	UserID string
	Notes  []Note
}

type Note struct {
	Login       string
	Password    string
	ServiceName string
}

type BotHandler interface {
	WorkerFunc(updates <-chan tgbotapi.Update)
	Set(update tgbotapi.Update) (string, error)
	Get(update tgbotapi.Update) (string, error)
	Del(update tgbotapi.Update) (string, error)
}
```

# Телеграм бот для хранения паролей
Выполнил Гапонов Н. в качестве тестового задания к вакансии https://internship.vk.company/vacancy/626

## Описание задачи
Реализовать Telegram бота, обладающего функционалом персонального хранилища паролей.  
Поддерживаются следующие команды:
1. /set – добавить логин и пароль к сервису
2. /get – получить логин и пароль по названию сервиса
3. /del – удалить значения для сервиса

Выполнены основные требования к реализации:
1. Бот написан на Golang
2. Сообщения с паролями удаляются по истечении заданного константой времени
3. Каждому **чату** выводятся исключительно его пароли (для этого учитывается ID чата telegram). Можно было учесть ID
пользователя, но мне показалось логичнее разделить пространство заметок на основании чатов. В случае общения с ботом
напрямую (userID == chatID), бот будет хранить личные данные. При добавлении в чат, бот будет хранить данные, общие
для участников чата.

Также выполнены дополнительные требования к реализации:
1. ~Надеюсь, что~ контейнер запущен на 87.239.111.2, а сам бот доступен по ссылке http://t.me/password_keeper_by_booec_bot
2. Можно запустить бота при помощи `docker compose up`
3. Для хранения данных используется MongoDB.

## Описание решения
```
├── build
│   └── Dockerfile
├── cmd
│   └── tgbot
│       ├── main.go — Инициализация логгера, хендлера БД и самого бота, запуск воркеров (функций, обрабатывающих запросы пользователя).
│       └── secret.go — Файл не импортирован в git, здесь хранится ключ бота.
├── docker-compose.yml
├── go.mod
├── go.sum
├── internal
    ├── handler
    │   ├── handler.go — Реализация handler'а: Предоставлен метод `HandleUpdates`, который может вызываться несколько раз (можно использовать worker pool).
    │   └── interface.go — Интерфейс handler'а, используемый main'ом.
    └── repo
        ├── interface.go — Интерфейс репозитория, используемый handler'ом.
        └── mongorepo.go — Реализация репозитория с использованием MongoDB.
```
**Примечание:** Для локального запуска в `./cmd/tgbot/main.go` замените `WebhookURL`, а также задайте `BotToken` (в целях безопасности он не импортирован в git-репозиторий).


*Далее идёт вольное описание кода*  
При реализации хендлера, я решил продумать возможность обрабатывать запросы многопоточно. Заметим, что `tgbotapi.UpdatesChannel`
– канал апдейтов, в связи с чем можно применить Worker Pool: создать несколько горутин (их количество регулируется константой),
которые извлекают данные из канала и обрабатывают запросы независимо. Очевидно, что конкретное значение из канала сможет считать
только одна горутина, даже если свободных горутин несколько.

Сначала я описал необходимые мне структуры, интерфейсы хендлера и репозитория:  

tgbot/internal/repo/interface.go:
```golang
type Workspace struct {
	ChatID string
	Notes  []Note
}

type Note struct {
	ServiceName string
	Login       string
	Password    string
}

var (
	ErrNotFound = errors.New("not found")
)

type NotesRepo interface {
	Set(userID string, note Note) error
	Get(userID, serviceName string) (Note, error)
	Del(userID, serviceName string) error
}
```

tgbot/internal/handler/interface.go:
```golang
type BotHandler interface {
	// HandleUpdates parses update and run command handler func as new goroutine.
	// You can have many HandleUpdates running simultaneously.
	HandleUpdates(updates <-chan tgbotapi.Update)

	Set(msg *tgbotapi.Message) (string, error)
	Get(msg *tgbotapi.Message) (string, error)
	Del(msg *tgbotapi.Message) (string, error)
}
```

На самом деле main.go не использует BotHandler.Set(), BotHandler.Get() и BotHandler.Del(), однако мне показалось корректным
описать эти функции в интерфейсе для лучшего понимания контекста.  

Далее я начал реализовывать BotHandler. Кажется логичнее было начать с репозитория, но интерфейс позволял мне просто
вызывать repo.Set()/Get()/Del(), на тот момент не задумываясь об их реализации.

Сам экземпляр воркера (`HandleUpdates`) достаточно прост: он читает из канала, предоставленного API телеграма, извлекает из
сообщения команду (`msg.Command()` позволяет получить текст первого слова, если оно начинается с "/") и вызывает как отдельную
горутину функцию-обработчик. Немного позже я сделал некое подобие middleware'а, обернув вызов функции `handlerFunc` подсчётом
времени её работы, выводом логов и оформлением ответа пользователю:
```golang
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
```

Сами функции-хендлеры можно получить из словаря `Handler.CommandHandlers map[string]func(msg *tgbotapi.Message) (string, error)`, что
позволяет одновременно проверить ввод пользователя на валидность и получить необходимую функцию-обработчик. Реализация функций Handler.Get(),
.Set() и .Del() достаточно очевидна, единственным интересным моментом является обработка имени сервиса, состоящего из нескольких слов.

Самым интересным же оказалось написать MongoRepo, который и будет обеспечивать нам взаимодействие с MongoDB. За свой небольшой опыт, я чаще работал
с SQL-базами, поэтому сейчас решил попрактиковаться с Mongo. В целом сами запросы тривиальны: в коллеции хранится по документу на каждый чат.
Документ представляет из себя \_id (ID чата по совместительству является уникальным ID документа в Mongo) и notes — массив из записей (названий сервисов,
логинов и паролей).

При реализации я столкнулся с небольшой проблемой: Если Get и Del можно было реализовать в один поход к БД, тем самым обеспечив безопасность
многопоточности, то в Set() необходимо было сделать несколько запросов: первым очистить прошлые данные о логине-пароле для сервиса, а вторым запросом
уже записать новые данные. Чтобы гарантировать валидную работу при многопоточности, я решил реализовать NamedMutexes: структуру, которая бы смогла
управлять "именованными мьютексами" и предоставлять такой интерфейс:
1. `.Lock(mutexName string)` — закрыть мьютекс с именем `mutexName`.
2. `.Unlock(mutexName string)` — открыть мьютекс с именем `mutexName`.

*Этот код не вошёл в финальную версию проекта:*
```golang
type NamedMutexes struct {
	sync.Map
}

func (m *NamedMutexes) Lock(mutexName string) error {
	value, _ := m.LoadOrStore(mutexName, &sync.Mutex{})

	mu, ok := value.(*sync.Mutex)
	if mu == nil || !ok {
		return errors.New("got not *sync.Mutex from map")
	}

	mu.Lock()
	return nil
}

func (m *NamedMutexes) Unlock(mutexName string) error {
	value, loaded := m.Load(mutexName)
	if !loaded {
		return errors.New("mutex not found in map")
	}

	mu, ok := value.(*sync.Mutex)
	if mu == nil || !ok {
		return errors.New("got not *sync.Mutex from map")
	}

	mu.Unlock()
	return nil
}
```

После написания данного кода, я решил разобраться с такой проблемой: в оперативной памяти бы хранилось слишком много ненужных мьютексов
(ведь пользователь может не использовать бота длительное время после добавления пароля). Так, я решил сделать автоматическое удаление
неиспользуемых мьютексов, и пока гуглил нужную мне информацию, наткнулся на уже реализованный модуль `"github.com/yudai/nmutex"`, реализующий
необходимый функционал и предусматривающий автоудаление мьютексов. Я проанализировал его реализацию, и заменил им свой модуль.

Параллельно с написанием go-кода, я также иногда корректировал Dockerfile и docker-compose.yml. В конечном счёте, это позволило через `scp`
скопировать файлы проекта на удалённую машину и запустить сервис через `docker compose up`.

### Недочёты (что можно добавить в проект)

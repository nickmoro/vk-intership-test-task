package main

import (
	"log"

	"tgbot/internal/handler"
	"tgbot/internal/repo"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	NumberOfWorkers = 4
	MongoURI        = "mongodb://mongodb:27017"
	DatabaseName    = "tgbot"
	CollectionName  = "users"
)

func main() {
	// zapSugaredLogger
	config := zap.NewDevelopmentConfig()
	// config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapLogger, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}
	logger := zapLogger.Sugar()
	defer func() {
		syncErr := logger.Sync()
		if syncErr != nil {
			log.Fatal(syncErr)
		}
	}()
	logger.Info("zapSugaredLogger initialized")

	// MongoDB
	repo, err := repo.NewMongoRepo(MongoURI, DatabaseName, CollectionName)
	if err != nil {
		logger.Panic(err)
	}
	logger.Info("MongoDB connection established")

	// tgbot
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		logger.Panic(err)
	}

	_, err = bot.RemoveWebhook()
	if err != nil {
		logger.Panic(err)
	}

	updates, err := bot.GetUpdatesChan(tgbotapi.NewUpdate(0))
	if err != nil {
		logger.Panic(err)
	}
	logger.Infof(`Got UpdatesChannel from bot "%v"`, bot.Self.UserName)

	botHandler := handler.NewBotHandler(logger, bot, repo)

	for i := 0; i < NumberOfWorkers; i++ {
		go func(workerNum int) {
			logger.Infof(`Worker %v started`, workerNum)
			botHandler.HandleUpdates(updates)
		}(i)
	}

	// Sleep forever
	<-make(chan int)
}

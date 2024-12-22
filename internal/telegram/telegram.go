package telegram

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"hosting/internal/global"
)

func InitTelegram() {
	var err error
	global.Bot, err = tgbotapi.NewBotAPI(global.AppConfig.Telegram.Token)
	if err != nil {
		log.Fatal(err)
	}
}

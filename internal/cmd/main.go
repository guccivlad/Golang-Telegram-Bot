package main

import (
	"bot/internal/config"
	"bot/internal/fetcher"
	"bot/internal/notifier"
	"bot/internal/storage"
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type ViewFunc func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error

func ViewCmdStart() ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		if _, err := bot.Send(tgbotapi.NewMessage(update.FromChat().ID, "Hello")); err != nil {
			return err
		}

		return nil
	}
}

type Bot struct {
	api      *tgbotapi.BotAPI
	cmdViews map[string]ViewFunc
}

func New(api *tgbotapi.BotAPI) *Bot {
	return &Bot{api: api}
}

func (b *Bot) RegisterCmdView(cmd string, view ViewFunc) {
	if b.cmdViews == nil {
		b.cmdViews = make(map[string]ViewFunc)
	}

	b.cmdViews[cmd] = view
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	// перехватываем панику в FuncView
	defer func() {
		p := recover()

		if p != nil {
			log.Println("ERROR: panic in ViewFunc recovered")
		}
	}()

	if update.Message == nil || !update.Message.IsCommand() {
		return
	}

	var view ViewFunc

	if !update.Message.IsCommand() {
		return
	}

	command := update.Message.Command()

	commandView, ok := b.cmdViews[command]

	if !ok {
		return
	}

	view = commandView

	viewErr := view(ctx, b.api, update)

	if viewErr != nil {
		log.Println("ERROR: execute view fail")

		_, sendErr := b.api.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Internal error"))

		if sendErr != nil {
			log.Printf("ERROR: failed to send error message")
		}
	}

}

func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			b.handleUpdate(updateCtx, update)
			updateCancel()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func main() {
	botAPI, err := tgbotapi.NewBotAPI(config.Get().TelegramBotToken)
	if err != nil {
		log.Println("ERROR: failed to create botAPI")
		return
	}

	db, err := sqlx.Connect("postgres", config.Get().DatabaseDSN)
	if err != nil {
		log.Printf("ERROR: failed to connect to db %v", err)
		return
	}
	defer db.Close()

	var (
		articleStorage = storage.NewArticleStorage(db)
		sourceStorage  = storage.NewSourceStorage(db)
		fetcher        = fetcher.New(
			articleStorage,
			sourceStorage,
			config.Get().FilterKeywords,
			config.Get().FetchInterval,
		)
		notifier = notifier.New(
			articleStorage,
			botAPI,
			config.Get().NotificationInterval,
			2*config.Get().FetchInterval,
			config.Get().TelegramChannelID,
		)
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	bot := New(botAPI)
	bot.RegisterCmdView("start", ViewCmdStart())

	go func(ctx context.Context) {
		if err := notifier.Run(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Println("ERROR: failed to run notifier")
				return
			}

			log.Println("Notifier has stopped")
		}
	}(ctx)

	go func(ctx context.Context) {
		if err := fetcher.Run(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Println("ERROR: failed to run fetcher")
				return
			}

			log.Println("Fetcher has stopped")
		}
	}(ctx)

	if err := bot.Run(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Println("ERROR: failed to run bot")
			return
		}

		log.Println("Bot has stopped")
	}
}

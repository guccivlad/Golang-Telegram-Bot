package notifier

import (
	"bot/internal/model"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ArticleProvider interface {
	AllNotPosted(ctx context.Context, since time.Time, limit uint64) ([]model.Article, error)
	MarkAsPosted(ctx context.Context, id int64) error
}

// type Summarizer interface {
// 	Summarize(text string) (string, error)
// }

type Notifier struct {
	articles ArticleProvider
	// summarizer       Summarizer
	bot              *tgbotapi.BotAPI
	sendInterval     time.Duration
	lookupTimeWindow time.Duration
	channelID        int64
}

func New(articleProvider ArticleProvider /*summarizer Summarizer,*/, bot *tgbotapi.BotAPI,
	sendInterval time.Duration, lookupTimeWindow time.Duration, channelID int64) *Notifier {

	return &Notifier{
		articles: articleProvider,
		// summarizer:       summarizer,
		bot:              bot,
		sendInterval:     sendInterval,
		lookupTimeWindow: lookupTimeWindow,
		channelID:        channelID,
	}
}

func (n *Notifier) Run(ctx context.Context) error {
	ticker := time.NewTicker(n.sendInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := n.SelectAndSendArticle(ctx); err != nil {
				return err
			}
		}
	}
}

func (n *Notifier) GetSummary(article model.Article) (string, error) {
	var r io.Reader

	if article.Summury != "" {
		r = strings.NewReader(article.Summury)
	} else {
		resp, err := http.Get(article.Link)

		if err != nil {
			return "", err
		}

		r = resp.Body

		resp.Body.Close()
	}

	text, err := readability.FromReader(r, nil)

	if err != nil {
		return "", err
	}

	// summary, summaryErr := n.summarizer.Summarize(text.TextContent)

	// if summaryErr != nil {
	// 	return "", summaryErr
	// }

	return text.TextContent, nil
}

func (n *Notifier) SelectAndSendArticle(ctx context.Context) error {
	articles, err := n.articles.AllNotPosted(ctx, time.Now().Add(-n.lookupTimeWindow), 1)

	if err != nil {
		return nil
	}

	if len(articles) == 0 {
		return nil
	}

	article := articles[0]

	summary, summaryErr := n.GetSummary(article)

	if summaryErr != nil {
		return summaryErr
	}

	sendErr := n.SendArticle(article, summary)

	if sendErr != nil {
		return sendErr
	}

	return n.articles.MarkAsPosted(ctx, article.ID)
}

func (n *Notifier) SendArticle(article model.Article, summary string) error {
	const msgFormat = "*%s*%s\n\n%s"

	msg := tgbotapi.NewMessage(n.channelID, fmt.Sprintf(
		msgFormat,
		EscapeForMarkdown(article.Title),
		EscapeForMarkdown(summary),
		EscapeForMarkdown(article.Link),
	))
	msg.ParseMode = "MarkdownV2"

	_, err := n.bot.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

var (
	replacer = strings.NewReplacer(
		"-",
		"\\-",
		"_",
		"\\_",
		"*",
		"\\*",
		"[",
		"\\[",
		"]",
		"\\]",
		"(",
		"\\(",
		")",
		"\\)",
		"~",
		"\\~",
		"`",
		"\\`",
		">",
		"\\>",
		"#",
		"\\#",
		"+",
		"\\+",
		"=",
		"\\=",
		"|",
		"\\|",
		"{",
		"\\{",
		"}",
		"\\}",
		".",
		"\\.",
		"!",
		"\\!",
	)
)

func EscapeForMarkdown(src string) string {
	return replacer.Replace(src)
}

package nocino

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kipters/nocino/pkg/sticker"

	"github.com/kipters/nocino/pkg/gif"
	"github.com/kipters/nocino/pkg/markov"

	"github.com/sirupsen/logrus"
	"gopkg.in/telegram-bot-api.v4"
)

type Nocino struct {
	API            *tgbotapi.BotAPI
	BotUsername    string
	Numw           int
	Plen           int
	GIFmaxsize     int
	Stickermaxsize int
	TrustedMap     map[int]bool
	Log            *logrus.Entry
}

func NewNocino(tgtoken string, trustedIDs string, numw int, plen int, gifmaxsize int, stickermaxsize int, logger *logrus.Logger) *Nocino {
	trustedMap := make(map[int]bool)
	if trustedIDs != "" {
		ids := strings.Split(trustedIDs, ",")
		for i := 0; i < len(ids); i++ {
			j, _ := strconv.Atoi(ids[i])
			trustedMap[j] = true
		}
	}
	logfields := logger.WithField("component", "nocino")

	bot, err := tgbotapi.NewBotAPI(tgtoken)
	if err != nil {
		logfields.Fatal("Cannot log in, exiting...")
	}
	botUsername := fmt.Sprintf("@%s", bot.Self.UserName)
	logfields.Infof("Authorized on account %s", botUsername)

	return &Nocino{
		API:            bot,
		BotUsername:    botUsername,
		Numw:           numw,
		Plen:           plen,
		GIFmaxsize:     gifmaxsize,
		Stickermaxsize: stickermaxsize,
		TrustedMap:     trustedMap,
		Log:            logfields,
	}
}

func (n *Nocino) RunStatsTicker(markov *markov.Chain, gifdb *gif.GIFDB, stickerdb *sticker.STICKERDB) {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			n.Log.Infof("Nocino Stats: %d Markov suffixes, %d GIF in Database, %d stickers",
				len(markov.Chain), len(gifdb.List), len(stickerdb.List))
		}
	}()
}

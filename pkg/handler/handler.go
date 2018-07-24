package handler

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/kipters/nocino/pkg/sticker"

	"github.com/davecgh/go-spew/spew"

	"github.com/kipters/nocino/pkg/gif"
	"github.com/kipters/nocino/pkg/markov"
	"github.com/kipters/nocino/pkg/nocino"
	"github.com/sirupsen/logrus"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type Handler struct {
	nocino    *nocino.Nocino
	update    tgbotapi.Update
	markov    *markov.Chain
	gifdb     *gif.GIFDB
	stickerdb *sticker.STICKERDB
	log       *logrus.Entry
}

func NewHandler(nocino *nocino.Nocino, update tgbotapi.Update, markov *markov.Chain, gifdb *gif.GIFDB, stickerdb *sticker.STICKERDB) *Handler {
	loghandler := nocino.Log.WithFields(logrus.Fields{
		"username":  update.Message.From.UserName,
		"subsystem": "handler",
	})

	return &Handler{
		nocino:    nocino,
		update:    update,
		markov:    markov,
		gifdb:     gifdb,
		stickerdb: stickerdb,
		log:       loghandler,
	}
}

func (h *Handler) Handle() error {
	var answerRequired = false
	var tokens []string

	h.log.Debugf("Incoming message: %#v", spew.Sdump(h.update))

	switch h.update.Message.Chat.Type {
	case "private":
		if ok := h.checkTrustedID(h.update.Message.From.ID); !ok {
			return nil
		}
		answerRequired = true
		fallthrough
	default:
		answerRequired, tokens = h.processMessage()
	}

	defer h.saveMessage(tokens)

	if answerRequired {
		dice := h.rollDice()

		if dice > 5 && len(h.stickerdb.List) > 0 {
			h.nocino.API.Send(h.fetchSticker())
			return nil
		}

		if dice > 4 && len(h.gifdb.List) > 0 {
			h.nocino.API.Send(h.fetchGIF())
			return nil
		}
		if len(h.markov.Chain) > 0 {
			h.nocino.API.Send(h.genText())
			return nil
		}
		h.log.Warnf("Not replying :-( Markov chain is empty")
	}

	return nil

}

func (h *Handler) rollDice() int {
	dice := []int{1, 2, 3, 4, 5, 6}
	rand.Seed(time.Now().UnixNano())
	return dice[rand.Intn(len(dice)-1)]
}

func (h *Handler) genText() tgbotapi.Chattable {
	// Generate a Markov Chain
	genText, elapsed := h.markov.GenerateChain(h.nocino.Numw, h.update.Message.Text)
	h.log.WithField("elapsed", elapsed.String()).Infof("Sending response: '%s'", genText)
	// Compose message
	msg := tgbotapi.NewMessage(h.update.Message.Chat.ID, genText)
	msg.ReplyToMessageID = h.update.Message.MessageID

	return msg
}

func (h *Handler) fetchGIF() tgbotapi.Chattable {
	gifpick := fmt.Sprintf("%s/%s", h.gifdb.Store, h.gifdb.GetRandom())
	h.log.Infof("Sending GIF: %s", gifpick)
	msg := tgbotapi.NewDocumentUpload(h.update.Message.Chat.ID, gifpick)
	msg.ReplyToMessageID = h.update.Message.MessageID

	return msg
}

func (h *Handler) fetchSticker() tgbotapi.Chattable {
	stickerpick := fmt.Sprintf("%s/%s", h.stickerdb.Store, h.stickerdb.GetRandom())
	h.log.Infof("Sending sticker: %s", stickerpick)
	msg := tgbotapi.NewDocumentUpload(h.update.Message.Chat.ID, stickerpick)
	msg.ReplyToMessageID = h.update.Message.MessageID

	return msg
}

func (h *Handler) saveMessage(tokens []string) {
	if len(tokens) > 0 {
		// add message to chain
		h.log.Debugf("Saving tokens to Chain '%v'", tokens)
		h.markov.AddChain(strings.Join(tokens, " "))
	}

	if h.update.Message.Sticker != nil && h.update.Message.Sticker.FileSize < h.nocino.Stickermaxsize {
		if err := h.stickerdb.Hoard(h.update, h.nocino.API); err != nil {
			h.log.Errorf("Could not save sticker due to error '%s'", err)
			return
		}
		h.log.Debugf("Saving sticker to DB '%s'", h.update.Message.Sticker.FileID)
		h.stickerdb.Add(fmt.Sprintf("%s.webm", h.update.Message.Sticker.FileID))
	}

	if h.update.Message.Document != nil && (h.update.Message.Document.MimeType == "video/mp4" && h.update.Message.Document.FileSize < h.nocino.GIFmaxsize) {
		if err := h.gifdb.Hoard(h.update, h.nocino.API); err != nil {
			h.log.Errorf("Could not save GIF due to error '%s'", err)
			return
		}
		h.log.Debugf("Saving GIF to DB '%s'", h.update.Message.Document.FileID)
		h.gifdb.Add(fmt.Sprintf("%s.mp4", h.update.Message.Document.FileID))
	}

}

func (h *Handler) checkTrustedID(userid int) bool {
	if h.nocino.TrustedMap[userid] {
		h.log.Infof("Authorized private chat, asking: '%s'", h.update.Message.Text)
		return true
	}
	// if it's not in the authorized list, do not log
	h.log.Warnf("Unauthorized private chat, asking: '%s'", h.update.Message.Text)
	return false
}

func (h *Handler) processMessage() (answerRequired bool, tokens []string) {
	// tokenize message
	tokens = strings.Split(h.update.Message.Text, " ")

	// if it's a reply, check if it's to us, answer back if necessary.
	if h.update.Message.ReplyToMessage != nil && h.update.Message.ReplyToMessage.From.UserName == h.nocino.API.Self.UserName {
		h.log.Infof("Reply to us, asking: '%s'", h.update.Message.Text)
		answerRequired = true
	}

	// check if we're being mentioned, answer back if necessary.
	if strings.ToLower(tokens[0]) == strings.ToLower(h.nocino.BotUsername) {
		// pop the first element
		tokens = tokens[1:]
		h.log.Infof("Mention to us, asking: '%s'", strings.Join(tokens, " "))
		answerRequired = true
	}

	return answerRequired, tokens
}

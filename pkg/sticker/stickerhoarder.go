package sticker

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/telegram-bot-api.v4"
)

type STICKERDB struct {
	List  []string
	Store string
	mutex sync.Mutex
	log   *logrus.Entry
}

func NewSTICKERDB(stickerstore string, logger *logrus.Logger) *STICKERDB {
	logfields := logger.WithField("component", "stickerdb")
	return &STICKERDB{
		Store: stickerstore,
		log:   logfields,
	}
}

func (s *STICKERDB) Add(in string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.List = append(s.List, in)

	return nil
}

func (s *STICKERDB) GetRandom() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.List[rand.Intn(len(s.List))]

}

func (s *STICKERDB) ReadList() {
	// Create stickerstore if doesn't exist
	if _, err := os.Stat(s.Store); os.IsNotExist(err) {
		s.log.Warnf("Directory '%s' does not exist, creating...", s.Store)
		if err := os.Mkdir(s.Store, 0700); err != nil {
			s.log.Fatalf("Cannot create directory '%s', exiting", s.Store)
		}
	}

	stickers, err := ioutil.ReadDir(s.Store)
	if err != nil {
		return
	}

	for _, file := range stickers {
		s.List = append(s.List, file.Name())
	}
	s.log.Infof("Loaded sticker list from '%s' (%d element/s).", s.Store, len(s.List))
}

func (g *STICKERDB) Hoard(update tgbotapi.Update, bot *tgbotapi.BotAPI) error {
	g.log.WithFields(logrus.Fields{
		"username": update.Message.From.UserName,
	}).Infof("Hoarding sticker ID '%s'", update.Message.Sticker.FileID)
	file, err := bot.GetFileDirectURL(update.Message.Sticker.FileID)
	if err != nil {
		return err
	}
	out, err := os.Create(fmt.Sprintf("%s/%s.webp", g.Store, update.Message.Sticker.FileID))
	if err != nil {
		g.log.Errorf("Can't open file '%s' for writing", fmt.Sprintf("%s/%s.webp", g.Store, update.Message.Sticker.FileID))
		return err
	}
	defer out.Close()
	resp, err := http.Get(file)
	if err != nil {
		g.log.Errorf("Can't fetch file '%s' from telegram", file)
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

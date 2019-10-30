package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/rylio/ytdl"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
)

var Bot *tgbotapi.BotAPI
var YoutubeLinkRegExp = regexp.MustCompile(`.*(((?:https?:)?\/\/)?((?:www|m)\.)?((?:youtube\.com|youtu.be))(\/(?:[\w\-]+\?v=|embed\/|v\/)?)([\w\-]+)(\S+)?).*`)

func init() {
	var err error
	Bot, err = tgbotapi.NewBotAPI(os.Getenv("YTB_BOT_API"))
	if err != nil {
		log.Panic(err)
	}

	//Bot.Debug = true

	log.Printf("Authorized on account %s", Bot.Self.UserName)
}

type Response struct {
	ChatID int64
	fileInfo *FileInfo
}

const BotAgentChatID = 846525283

func BotMainLoop() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := Bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	var lastInfoMsg tgbotapi.Message

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}
		if update.Message.IsCommand() {
			if update.Message.Command() == "ping" {
				Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "pong"))
				continue
			}
			if update.Message.Command() == "start" {
				continue
			}
		}

		// client sended to us video
		if update.Message.Video != nil && update.Message.Chat.ID == BotAgentChatID {
			caption := strings.Split(update.Message.Caption, ":")
			chatIDStr := caption[0]
			fileName := caption[1]
			chatID, _ := strconv.ParseInt(chatIDStr, 10, 64)

			// delete old info msg
			_, err = Bot.DeleteMessage(tgbotapi.NewDeleteMessage(chatID, lastInfoMsg.MessageID))
			if err != nil {
				log.Error(err)
			}

			ShareVideoFile(chatID, update.Message.Video)
			// remove old video file
			Storage.RemoveFile(fileName)
			continue
		}

		if YoutubeLinkRegExp.MatchString(update.Message.Text) == false {
			Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Bad url format"))
			continue
		}

		lastInfoMsg, err = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Uploading..."))

		log.Info("Url is ", update.Message.Text)

		go func(text string, chatID int64) {
			err = RequestHanlder(text, chatID)
			if err != nil {
				log.Error(err)
				Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Failed download video"))
			}
		}(YoutubeURLFromText(update.Message.Text), update.Message.Chat.ID)
	}
}

func RequestHanlder(messageTxt string, fromChatID int64) error {
	vid, err := ytdl.GetVideoInfo(messageTxt)
	if err != nil {
		return errors.WithMessage(err, "failed get video info")
	}

	format720p, format360p := GetBestVideoFormats(vid.Formats)

	fileInfo, err := SaveVideo(vid, format720p, format360p)
	if err != nil {
		return errors.WithMessage(err, "failed save video")
	}

	log.Info("Video downloaded")
	ResponseChan <- &Response{fromChatID, fileInfo}

	return nil
}

func ShareVideoFile(chatID int64, video *tgbotapi.Video) {
	log.Info("share video")
	vidShare := tgbotapi.NewVideoShare(chatID, video.FileID)
	_, err := Bot.Send(vidShare)
	if err != nil {
		log.Error(err)
	}
}

func YoutubeURLFromText(text string) string {
	text = strings.ReplaceAll(text,"\t", "")
	text = strings.ReplaceAll(text,"\n", "")
	text = strings.ReplaceAll(text,"\v", "")
	text = strings.ReplaceAll(text,"\f", "")
	text = strings.ReplaceAll(text,"\r","")

	return "https://" + YoutubeLinkRegExp.ReplaceAllString(text, "$1")
}
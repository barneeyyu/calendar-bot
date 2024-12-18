package main

import (
	"bytes"
	"errors"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/sirupsen/logrus"
)

const (
	SEVERITY  = "severity"
	MESSAGE   = "message"
	TIMESTAMP = "timestamp"
)

type EnvVars struct {
	botClient *linebot.Client
}

func getEnvironmentVariables() (envVars *EnvVars, err error) {
	channelSecret := os.Getenv("CHANNEL_SECRET")
	if channelSecret == "" {
		return nil, errors.New("CHANNEL_SECRET is not set")
	}

	channelToken := os.Getenv("CHANNEL_TOKEN")
	if channelToken == "" {
		return nil, errors.New("CHANNEL_TOKEN is not set")
	}

	// 初始化 LINE Bot
	bot, err := linebot.New(
		channelSecret,
		channelToken,
	)
	if err != nil {
		return nil, errors.New("initial line bot failed")
	}

	return &EnvVars{
		botClient: bot,
	}, nil

}

func Handler(request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  TIMESTAMP,
			logrus.FieldKeyLevel: SEVERITY,
			logrus.FieldKeyMsg:   MESSAGE,
		},
	})

	logger.Infof("Processing Lambda request %s\n", request.RequestContext.RequestID)

	envVars, err := getEnvironmentVariables()
	if err != nil {
		logger.Errorf("get environment variables failed: %s", err.Error())
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	// 測試 bot client
	botInfo, err := envVars.botClient.GetBotInfo().Do()
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	logger.Infof("bot name: %s", botInfo.DisplayName)
	logger.Infof("message: %s", request.Body)

	// 解析 LINE Webhook 事件
	reqBody := bytes.NewBufferString(request.Body)
	req, err := http.NewRequest("POST", "", reqBody)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	// 複製所有 headers
	req.Header = make(http.Header)
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	messageEvents, err := envVars.botClient.ParseRequest(req)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			return events.APIGatewayProxyResponse{
				Body:       "Invalid signature",
				StatusCode: 400,
			}, nil
		}
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}
	// 處理事件
	for _, event := range messageEvents {
		logger.WithFields(logrus.Fields{
			"event_type": event.Type,
			"user_id":    event.Source.UserID,
			"room_id":    event.Source.RoomID,
			"group_id":   event.Source.GroupID,
		}).Info("event handling")
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				// 處理文字訊息
				logger.Infof("Received text message: %s", message.Text)

				// 回覆訊息
				if _, err = envVars.botClient.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("收到你的訊息："+message.Text)).Do(); err != nil {
					logger.Error("Error replying to message: ", err)
				}
			}
		}
	}

	return events.APIGatewayProxyResponse{
		Body:       "Hello World",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(Handler)
}

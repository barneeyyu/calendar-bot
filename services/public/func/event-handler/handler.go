package main

import (
	"bytes"
	"calendar-bot/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
)

const (
	timeFormat = "2006-01-02 15:04"
	timeZone   = "Asia/Taipei"
)

var (
	ErrPastDateTime   = errors.New("can't set past time")
	ErrFutureDateTime = errors.New("can't set future time over 1 year")
)

type Handler struct {
	logger       *logrus.Entry
	envVars      *EnvVars
	openaiClient utils.OpenaiHandler
}

type ReminderEvent struct {
	UserID string `json:"userId"`
	Task   string `json:"task"`
}

func NewHandler(logger *logrus.Entry, envVars *EnvVars, openaiClient utils.OpenaiHandler) (*Handler, error) {
	return &Handler{
		logger:       logger,
		envVars:      envVars,
		openaiClient: openaiClient,
	}, nil
}

func (h *Handler) EventHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	h.logger.Infof("Processing Lambda request %s", request.RequestContext.RequestID)
	var bodyJSON interface{}
	if err := json.Unmarshal([]byte(request.Body), &bodyJSON); err != nil {
		h.logger.WithError(err).Error("Failed to parse JSON")
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	} else {
		h.logger.WithFields(logrus.Fields{
			"webhook_body": bodyJSON,
		}).Info("Received LINE webhook")
	}

	// analyze request body
	reqBody := bytes.NewBufferString(request.Body)
	req, err := http.NewRequest(http.MethodPost, "", reqBody)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	// parse all headers
	req.Header = make(http.Header)
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	messageEvents, err := h.envVars.botClient.ParseRequest(req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to parse request")
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
	// handle events
	for _, event := range messageEvents {
		h.logger.WithFields(logrus.Fields{
			"event_type": event.Type,
			"user_id":    event.Source.UserID,
			"room_id":    event.Source.RoomID,
			"group_id":   event.Source.GroupID,
		}).Info("event handling")
		if event.Type == linebot.EventTypeMessage {
			var replyMessage string
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				// handle text message
				h.logger.Infof("Received text message: %s", message.Text)

				validSchedule, err := h.openaiClient.TransferValidSchedule(message.Text)
				if err != nil {
					h.logger.WithError(err).Error("Failed to transfer valid schedule")
					return events.APIGatewayProxyResponse{
						Body:       err.Error(),
						StatusCode: 500,
					}, nil
				}
				h.logger.WithFields(logrus.Fields{
					"valid_schedule": validSchedule,
				}).Info("Transfer valid schedule from openai API")

				if validSchedule.Valid {
					utcScheduledTime, err := h.validateScheduleTimeAndTask(validSchedule.DateTime)
					if err != nil {
						h.logger.WithError(err).Error("Failed to handle text message")
						if err == ErrPastDateTime {
							replyMessage = "無法設定過去的時間"
						} else if err == ErrFutureDateTime {
							replyMessage = "無法設定未來超過一年的時間"
						} else {
							return events.APIGatewayProxyResponse{
								Body:       err.Error(),
								StatusCode: 500,
							}, nil
						}
					} else {
						// create reminder
						if err := h.createReminder(event.Source.UserID, utcScheduledTime, validSchedule.Task); err != nil {
							h.logger.WithError(err).Error("Failed to create reminder")
							replyMessage = "提醒設定失敗，請稍後再試"
						}
						replyMessage = fmt.Sprintf("提醒您：%s %s 已設定成功", validSchedule.DateTime, validSchedule.Task)
					}
				} else {
					replyMessage = "輸入格式錯誤，必須包含日期、時間、要做的事情。請重新輸入"
				}

				// reply message
				if _, err = h.envVars.botClient.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
					h.logger.WithError(err).Error("Error replying to message")
				}
			}
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func (h *Handler) validateScheduleTimeAndTask(dateTime string) (time.Time, error) {
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		return time.Time{}, err
	}
	scheduledTime, err := time.ParseInLocation(timeFormat, dateTime, loc)
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now().In(loc)

	// check if scheduled time is past
	if scheduledTime.Before(now) {
		return time.Time{}, ErrPastDateTime
	}

	// check if scheduled time is over 1 year
	oneYearLater := now.AddDate(1, 0, 0)
	if scheduledTime.After(oneYearLater) {
		return time.Time{}, ErrFutureDateTime
	}

	utcScheduledTime := scheduledTime.UTC()

	return utcScheduledTime, nil
}

func (h *Handler) createReminder(userID string, scheduledTime time.Time, task string) error {
	h.logger.WithFields(logrus.Fields{
		"original_time":       scheduledTime,
		"formatted_time":      scheduledTime.Format("2006-01-02T15:04:00"),
		"schedule_expression": fmt.Sprintf("at(%s)", scheduledTime.Format("2006-01-02T15:04:00")),
	}).Info("Creating schedule")

	// 準備要傳給 event-reminder 的資料
	reminderEvent := ReminderEvent{
		UserID: userID,
		Task:   task,
	}
	payload, err := json.Marshal(reminderEvent)
	if err != nil {
		return err
	}

	// create schedule
	_, err = h.envVars.schedulerClient.CreateSchedule(context.TODO(), &scheduler.CreateScheduleInput{
		Name: aws.String(fmt.Sprintf("reminder-%s-%d", userID, time.Now().Unix())),
		FlexibleTimeWindow: &types.FlexibleTimeWindow{
			Mode: types.FlexibleTimeWindowModeOff,
		},
		ScheduleExpression: aws.String(fmt.Sprintf("at(%s)", scheduledTime.Format("2006-01-02T15:04:00"))),
		Target: &types.Target{
			Arn:     aws.String(h.envVars.ReminderFunctionArn),
			RoleArn: aws.String(h.envVars.SchedulerRoleArn),
			Input:   aws.String(string(payload)),
		},
	})

	return err
}

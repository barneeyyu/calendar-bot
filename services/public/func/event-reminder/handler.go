package main

import (
	"context"
	"fmt"

	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	logger  *logrus.Entry
	envVars *EnvVars
}

type ReminderEvent struct {
	UserID string `json:"userId"`
	Task   string `json:"task"`
}

func NewHandler(logger *logrus.Entry, envVars *EnvVars) (*Handler, error) {
	return &Handler{
		logger:  logger,
		envVars: envVars,
	}, nil
}

func (h *Handler) EventHandler(ctx context.Context, event ReminderEvent) error {
	h.logger.Infof("Received reminder event for user %s: %s", event.UserID, event.Task)

	// send reminder message to user by using UserID with linebot
	mentionText := fmt.Sprintf("🔔提醒您：\n記得%s", event.Task)
	message := linebot.NewTextMessage(mentionText).
		WithSender(&linebot.Sender{
			Name: "Reminder Bot",
		})
	if _, err := h.envVars.botClient.PushMessage(event.UserID, message).Do(); err != nil {
		h.logger.WithError(err).Error("Failed to send reminder message")
		return err
	}
	h.logger.Info("Successfully sent reminder message")

	return nil
}

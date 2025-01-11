package main

import (
	"context"
	"fmt"

	"github.com/line/line-bot-sdk-go/linebot"
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

	// get user profile
	userProfile, err := h.envVars.botClient.GetProfile(event.UserID).Do()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user profile")
		return err
	}

	// send reminder message to user by using UserID with linebot
	mentionText := fmt.Sprintf("@%s\n%s", userProfile.DisplayName, event.Task)
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

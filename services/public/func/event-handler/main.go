package main

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/sirupsen/logrus"
)

const (
	SEVERITY    = "severity"
	MESSAGE     = "message"
	TIMESTAMP   = "timestamp"
	COMPONENT   = "component"
	SERVICENAME = "event-handler"
)

type EnvVars struct {
	botClient           *linebot.Client
	schedulerClient     *scheduler.Client
	ReminderFunctionArn string
	SchedulerRoleArn    string
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

	// initialize LINE Bot
	bot, err := linebot.New(
		channelSecret,
		channelToken,
	)
	if err != nil {
		return nil, errors.New("initial line bot failed")
	}

	// create EventBridge Scheduler client
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	schedulerClient := scheduler.NewFromConfig(cfg)

	reminderFunctionArn := os.Getenv("REMINDER_FUNCTION_ARN")
	if reminderFunctionArn == "" {
		return nil, errors.New("REMINDER_FUNCTION_ARN is not set")
	}

	schedulerRoleArn := os.Getenv("SCHEDULER_ROLE_ARN")
	if schedulerRoleArn == "" {
		return nil, errors.New("SCHEDULER_ROLE_ARN is not set")
	}

	return &EnvVars{
		botClient:           bot,
		schedulerClient:     schedulerClient,
		ReminderFunctionArn: reminderFunctionArn,
		SchedulerRoleArn:    schedulerRoleArn,
	}, nil
}

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  TIMESTAMP,
			logrus.FieldKeyLevel: SEVERITY,
			logrus.FieldKeyMsg:   MESSAGE,
		},
	})
	logger := logrus.WithField(COMPONENT, SERVICENAME)

	envVars, err := getEnvironmentVariables()
	if err != nil {
		logger.WithError(err).Error("Failed to get environment variables")
		panic(err)
	}

	handler, err := NewHandler(logger, envVars)
	if err != nil {
		logger.WithError(err).Error("Failed to create handler")
		panic(err)
	}

	lambda.Start(handler.EventHandler)
}

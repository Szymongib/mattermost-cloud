package webhook

import (
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
	"time"
)

type Sender struct {
	store       webhookStore
	environment string
}

func (s *Sender) SendInstallationWebhook(installation *model.Installation, oldState, newState string, logger log.FieldLogger) {
	if oldState == "" {
		oldState = "n/a"
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  newState,
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"DNS": installation.DNS, "Environment": s.environment},
	}

	err := SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}
}

//func (s *Sender) SendInstallationBackupWebhook(installation *model.Installation, oldState, newState string, logger log.FieldLogger)  {
//	webhookPayload := &model.WebhookPayload{
//		Type:      model.TypeInstallation,
//		ID:        installation.ID,
//		NewState:  newState,
//		OldState:  installation.State,
//		Timestamp: time.Now().UnixNano(),
//		ExtraData: map[string]string{"DNS": installation.DNS, "Environment": s.environment},
//	}
//
//	err := SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
//	if err != nil {
//		logger.WithError(err).Error("Unable to process and send webhooks")
//	}
//}

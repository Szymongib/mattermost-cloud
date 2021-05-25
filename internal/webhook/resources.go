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

func (s *Sender) SendInstallationWebhook(installation *model.Installation, oldState string, logger log.FieldLogger) {
	oldState = ensureNotEmptyState(oldState)

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  installation.State,
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"DNS": installation.DNS, "Environment": s.environment},
	}

	err := SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}
}

func (s *Sender) SendInstallationBackupWebhook(backup *model.InstallationBackup, oldState string, logger log.FieldLogger) {
	oldState = ensureNotEmptyState(oldState)

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationBackup,
		ID:        backup.ID,
		NewState:  string(backup.State),
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Installation": backup.InstallationID, "Environment": s.environment},
	}

	err := SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}
}

func (s *Sender) SendInstallationDBMigrationWebhook(migration *model.InstallationDBMigrationOperation, oldState string, logger log.FieldLogger) {
	oldState = ensureNotEmptyState(oldState)

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationDBMigration,
		ID:        migration.ID,
		NewState:  string(migration.State),
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Installation": migration.InstallationID, "Environment": s.environment},
	}

	err := SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}
}

func ensureNotEmptyState(state string) string {
	if state == "" {
		return "n/a"
	}
	return state
}

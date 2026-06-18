package managers

import (
	"encoding/json"
	"fmt"

	"github.com/Ashutoshbind15/ssh-chess/common"
)

func (dm *DataManager) AppendLiveGameSnapshot(payload common.LiveGamePayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal live game payload: %w", err)
	}

	var maxVersion int
	if err := dm.db.Model(&common.LiveGameEvent{}).
		Where("game_id = ?", payload.GameID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error; err != nil {
		return err
	}

	event := common.LiveGameEvent{
		GameID:  payload.GameID,
		Version: maxVersion + 1,
		Payload: data,
	}
	return dm.db.Create(&event).Error
}

func (dm *DataManager) LoadActiveGameSnapshots() ([]common.LiveGamePayload, error) {
	var events []common.LiveGameEvent
	err := dm.db.Raw(`
		SELECT DISTINCT ON (game_id) *
		FROM live_game_events
		ORDER BY game_id, version DESC
	`).Scan(&events).Error
	if err != nil {
		return nil, err
	}

	var out []common.LiveGamePayload
	for _, event := range events {
		var payload common.LiveGamePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("unmarshal live game payload for %s: %w", event.GameID, err)
		}
		if payload.Status == GameStatusFinished {
			continue
		}
		out = append(out, payload)
	}
	return out, nil
}

func (dm *DataManager) DeleteLiveGameEvents(gameID string) error {
	return dm.db.Unscoped().Where("game_id = ?", gameID).Delete(&common.LiveGameEvent{}).Error
}

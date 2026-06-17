package managers

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Ashutoshbind15/ssh-chess/common"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DataManager struct {
	db *gorm.DB
}

func NewDataManager() *DataManager {
	dm := &DataManager{}
	dm.Init()
	return dm
}

func (dm *DataManager) Init() {
	dbURL := strings.TrimSpace(os.Getenv("DB_URL"))
	if dbURL == "" {
		panic("DB_URL environment variable is required")
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}

	if err := db.AutoMigrate(&common.Player{}, &common.Game{}, &common.BotGame{}); err != nil {
		panic(err)
	}

	dm.db = db
}

func (dm *DataManager) GetPlayer(fingerprint string) (*common.Player, error) {
	var player common.Player
	result := dm.db.First(&player, "fingerprint = ?", fingerprint)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &player, nil
}

func (dm *DataManager) AddPlayer(player common.Player) error {
	return dm.db.Create(&player).Error
}

const deletedPlayerUsername = "[deleted]"

// DeletePlayerData removes the player profile and bot games, and anonymizes
// the player's side of stored multiplayer game records.
func (dm *DataManager) DeletePlayerData(fingerprint string) error {
	if strings.TrimSpace(fingerprint) == "" {
		return fmt.Errorf("missing fingerprint")
	}

	return dm.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("player_fingerprint = ?", fingerprint).Delete(&common.BotGame{}).Error; err != nil {
			return err
		}

		if err := tx.Model(&common.Game{}).
			Where("white_fingerprint = ?", fingerprint).
			Updates(map[string]any{
				"white_fingerprint": "",
				"white_username":    deletedPlayerUsername,
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&common.Game{}).
			Where("black_fingerprint = ?", fingerprint).
			Updates(map[string]any{
				"black_fingerprint": "",
				"black_username":    deletedPlayerUsername,
			}).Error; err != nil {
			return err
		}

		return tx.Where("fingerprint = ?", fingerprint).Delete(&common.Player{}).Error
	})
}

func (dm *DataManager) AddGame(game common.Game) error {
	return dm.db.Create(&game).Error
}

func (dm *DataManager) GetGamesForPlayer(fingerprint string) ([]common.Game, error) {
	var games []common.Game
	result := dm.db.
		Where("white_fingerprint = ? OR black_fingerprint = ?", fingerprint, fingerprint).
		Order("created_at DESC").
		Find(&games)
	if result.Error != nil {
		return nil, result.Error
	}
	return games, nil
}

func (dm *DataManager) AddBotGame(game common.BotGame) error {
	return dm.db.Create(&game).Error
}

func (dm *DataManager) GetBotGamesForPlayer(fingerprint string) ([]common.BotGame, error) {
	var games []common.BotGame
	result := dm.db.
		Where("player_fingerprint = ?", fingerprint).
		Order("created_at DESC").
		Find(&games)
	if result.Error != nil {
		return nil, result.Error
	}
	return games, nil
}

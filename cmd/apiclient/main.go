package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Cludch/csgo-tools/internal/config"
	"github.com/Cludch/csgo-tools/internal/entity"
	"github.com/Cludch/csgo-tools/pkg/valveapi"
	"gorm.io/gorm"
)

var configData *config.Config
var db *gorm.DB

// Sets up the global variables (config, db) and the logger.
func init() {
	db = entity.GetDatabase()
	configData = config.GetConfiguration()

	configData.SetLoggingLevel()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		DisableColors: false,
	})
}

func main() {
	// Add accounts from config to database if not existing.
	// This also adds the first known share code.
	entity.AddConfigUsers(configData.CSGO)

	var csgoUsers []entity.CSGOUser

	// Create a loop that checks for new share codes each minute.
	t := time.NewTicker(time.Minute)
	for {
		result := db.Preload("ShareCode").Find(&csgoUsers, "match_history_authentication_code != ''")

		if err := result.Error; err != nil {
			panic(err)
		}

		// Iterate all csgo users and request the next share code for the latest share code.
		for _, csgoUser := range csgoUsers {
			if csgoUser.Disabled {
				continue
			}

			steamID := csgoUser.SteamID
			shareCode, err := valveapi.GetNextMatch(configData.Steam.SteamAPIKey, steamID, csgoUser.MatchHistoryAuthenticationCode, csgoUser.ShareCode.Encoded)

			// Disable user on error
			if err != nil {
				if os.IsTimeout(err) {
					log.Error("Lost connection", err)
					continue
				}
				db.Model(&csgoUser).Update("Disabled", true)
				log.Warnf("disabled csgo user %d due to an error in fetching the share code", steamID)
				log.Error(err)
				continue
			}

			// No new match.
			if shareCode == "" {
				log.Debugf("no new match found for %d", steamID)
				continue
			}

			log.Infof("found match share code %v", shareCode)

			// Create share code.
			sc := entity.CreateShareCodeFromEncoded(shareCode)
			// Create match.
			entity.CreateMatch(sc)
			// Update csgo user.
			csgoUser.UpdateLatestShareCode(sc)
		}
		<-t.C
	}
}

package schedule

import (
	"time"

	"github.com/go-co-op/gocron"
)

func Start() {
	s := gocron.NewScheduler(time.UTC)
	s.Every(1).Day().At("00:00").Do(deleteTempFile)
	s.Every(1).Day().At("00:00").Do(deleteTokens)
	s.StartAsync()
}

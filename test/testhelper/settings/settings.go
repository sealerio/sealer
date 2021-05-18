package settings

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

// init test params and settings
func init() {
	defaultWaiteTime := os.Getenv("DEFAULT_WAITE_TIME")
	if defaultWaiteTime == "" {
		DefaultWaiteTime = 300 * time.Second
	} else {
		DefaultWaiteTime, _ = time.ParseDuration(defaultWaiteTime)
	}

	maxWaiteTime := os.Getenv("MAX_WAITE_TIME")
	if maxWaiteTime == "" {
		MaxWaiteTime = 1200 * time.Second
	} else {
		MaxWaiteTime, _ = time.ParseDuration(maxWaiteTime)
	}

	pollingInterval := os.Getenv("DEFAULT_POLLING_INTERVAL")
	if pollingInterval == "" {
		DefaultPollingInterval = 10
	}
}

func GetTestImageName() string {
	return fmt.Sprintf("%s%d", ImageName, rand.Intn(99999))
}

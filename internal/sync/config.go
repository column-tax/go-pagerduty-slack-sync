package sync

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	scheduleKeyPrefix              = "SCHEDULE_"
	pagerDutyTokenKey              = "PAGERDUTY_TOKEN"
	slackTokenKey                  = "SLACK_TOKEN"
	runInterval                    = "RUN_INTERVAL_SECONDS"
	pdScheduleLookaheadKey         = "PAGERDUTY_SCHEDULE_LOOKAHEAD"
	runIntervalDefault             = 60
	syncAllOnCallGroup             = "SYNC_ALL_ONCALL_GROUP"
	allOnCallGroupHandlePrefix     = "ALL_ONCALL_GROUP_NAME_PREFIX"
	syncCurrentOnCallGroup         = "SYNC_CURRENT_ONCALL_GROUP"
	currentOnCallGroupHandlePrefix = "CURRENT_ONCALL_GROUP_NAME_PREFIX"
)

// Config is used to configure application
// PagerDutyToken - token used to connect to pagerduty API
// SlackToken - token used to connect to Slack API
type Config struct {
	Schedules                      []Schedule
	PagerDutyToken                 string
	SlackToken                     string
	RunIntervalInSeconds           int
	PagerdutyScheduleLookahead     time.Duration
	AllOncallGroupHandlePrefix     string
	SyncAllOncallGroup             bool
	CurrentOncallGroupHandlePrefix string
	SyncCurrentOncallGroup         bool
}

// Schedule models a PagerDuty schedule that will be synced with Slack
// ScheduleIDs - All PagerDuty schedule ID's to sync
// AllOnCallGroupName - Slack group name for all members of schedule
// CurrentOnCallGroupName - Slack group name for current person on call
type Schedule struct {
	ScheduleIDs              []string
	AllOnCallGroupHandle     string
	CurrentOnCallGroupHandle string
	SyncAllOnCallGroup       bool
	SyncCurrentOnCallGroup   bool
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

// NewConfigFromEnv is a function to generate a config from env varibles
// PAGERDUTY_TOKEN - PagerDuty Token
// SLACK_TOKEN - Slack Token
// SCHEDULE_XXX="id,name" e.g. 1234,platform-engineer will generate a schedule with the following values
// ScheduleID = "1234", AllOnCallGroupName = "all-oncall-platform-engineers", CurrentOnCallGroupName: "current-oncall-platform-engineer"
func NewConfigFromEnv() (*Config, error) {
	config := &Config{
		PagerDutyToken:                 os.Getenv(pagerDutyTokenKey),
		SlackToken:                     os.Getenv(slackTokenKey),
		RunIntervalInSeconds:           runIntervalDefault,
		AllOncallGroupHandlePrefix:     getEnv(allOnCallGroupHandlePrefix, ""),
		CurrentOncallGroupHandlePrefix: getEnv(currentOnCallGroupHandlePrefix, ""),
		SyncAllOncallGroup:             false,
		SyncCurrentOncallGroup:         true,
	}

	runInterval := os.Getenv(runInterval)
	v, err := strconv.Atoi(runInterval)
	if err == nil {
		config.RunIntervalInSeconds = v
	}

	syncAllOncallGroup := os.Getenv(syncAllOnCallGroup)
	b, err := strconv.ParseBool(syncAllOncallGroup)
	if err == nil {
		config.SyncAllOncallGroup = b
	}

	syncCurrentOncallGroup := os.Getenv(syncCurrentOnCallGroup)
	b, err = strconv.ParseBool(syncCurrentOncallGroup)
	if err == nil {
		config.SyncCurrentOncallGroup = b
	}

	pagerdutyScheduleLookahead, err := getPagerdutyScheduleLookahead()
	if err != nil {
		return nil, err
	}
	config.PagerdutyScheduleLookahead = pagerdutyScheduleLookahead

	for _, key := range os.Environ() {
		if strings.HasPrefix(key, scheduleKeyPrefix) {
			value := strings.Split(key, "=")[1]
			scheduleValues := strings.Split(value, ",")
			if len(scheduleValues) != 2 {
				return nil, fmt.Errorf("expecting schedule value to be a comma separated scheduleId,name but got %s", value)
			}

			config.Schedules = appendSchedule(config.Schedules, scheduleValues[0], scheduleValues[1], *config)
		}
	}

	if len(config.Schedules) == 0 {
		return nil, fmt.Errorf("expecting at least one schedule defined as an env var using prefix SCHEDULE_")
	}

	return config, nil
}

func appendSchedule(schedules []Schedule, scheduleID, teamName string, config Config) []Schedule {
	currentGroupHandle := fmt.Sprintf("%s%s", config.CurrentOncallGroupHandlePrefix, teamName)
	allGroupHandle := fmt.Sprintf("%s%ss", config.AllOncallGroupHandlePrefix, teamName)
	newScheduleList := make([]Schedule, len(schedules))
	updated := false

	for i, s := range schedules {
		if s.CurrentOnCallGroupHandle != currentGroupHandle {
			newScheduleList[i] = s

			continue
		}

		updated = true

		newScheduleList[i] = Schedule{
			ScheduleIDs:              append(s.ScheduleIDs, scheduleID),
			AllOnCallGroupHandle:     allGroupHandle,
			CurrentOnCallGroupHandle: currentGroupHandle,
			SyncAllOnCallGroup:       config.SyncAllOncallGroup,
			SyncCurrentOnCallGroup:   config.SyncCurrentOncallGroup,
		}
	}

	if !updated {
		newScheduleList = append(newScheduleList, Schedule{
			ScheduleIDs:              []string{scheduleID},
			AllOnCallGroupHandle:     allGroupHandle,
			CurrentOnCallGroupHandle: currentGroupHandle,
			SyncAllOnCallGroup:       config.SyncAllOncallGroup,
			SyncCurrentOnCallGroup:   config.SyncCurrentOncallGroup,
		})
	}

	return newScheduleList
}

func getPagerdutyScheduleLookahead() (time.Duration, error) {
	result := time.Hour * 24 * 100

	pdScheduleLookahead, ok := os.LookupEnv(pdScheduleLookaheadKey)
	if !ok {
		return result, nil
	}

	v, err := time.ParseDuration(pdScheduleLookahead)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s as time.Duration: %w", pdScheduleLookahead, err)
	}

	return v, nil
}

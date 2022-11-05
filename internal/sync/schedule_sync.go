package sync

import (
	"strings"
	"time"

	"github.com/kevholditch/go-pagerduty-slack-sync/internal/compare"
	"github.com/sirupsen/logrus"
)

// Schedules does the sync
func Schedules(config *Config) error {
	logrus.Infof("Running schedule sync...")
	s, err := newSlackClient(config.SlackToken)
	if err != nil {
		return err
	}
	p := newPagerDutyClient(config.PagerDutyToken)

	updateSlackGroup := func(emails []string, groupHandle string) error {
		slackIDs, err := s.getSlackIDsFromEmails(emails)
		if err != nil {
			return err
		}

		userGroup, err := s.createOrGetUserGroup(groupHandle)
		if err != nil {
			return err
		}
		members, err := s.Client.GetUserGroupMembers(userGroup.ID)
		if err != nil {
			return err
		}

		if !compare.Array(slackIDs, members) {
			logrus.Infof("Slack group %s needs updating...", groupHandle)
			_, err = s.Client.UpdateUserGroupMembers(userGroup.ID, strings.Join(slackIDs, ","))
			if err != nil {
				return err
			}
		}

		logrus.Infof("Slack group %s is up to date", groupHandle)
		return nil
	}

	getEmailsForSchedules := func(schedules []string, lookahead time.Duration) ([]string, error) {
		var emails []string

		for _, sid := range schedules {
			e, err := p.getEmailsForSchedule(sid, lookahead)
			if err != nil {
				return nil, err
			}

			emails = appendIfMissing(emails, e...)
		}

		return emails, nil
	}

	for _, schedule := range config.Schedules {

		if schedule.SyncCurrentOnCallGroup {
			logrus.Infof("Checking slack group: %s", schedule.CurrentOnCallGroupHandle)
			currentOncallEngineerEmails, err := getEmailsForSchedules(schedule.ScheduleIDs, time.Second)
			if err != nil {
				logrus.Errorf("Failed to get emails for %s: %v", schedule.CurrentOnCallGroupHandle, err)
				continue
			}

			err = updateSlackGroup(currentOncallEngineerEmails, schedule.CurrentOnCallGroupHandle)
			if err != nil {
				logrus.Errorf("Failed to update slack group %s: %v", schedule.CurrentOnCallGroupHandle, err)
				continue
			}

		}

		if schedule.SyncAllOnCallGroup {
			logrus.Infof("Checking slack group: %s", schedule.AllOnCallGroupHandle)

			allOncallEngineerEmails, err := getEmailsForSchedules(schedule.ScheduleIDs, config.PagerdutyScheduleLookahead)
			if err != nil {
				logrus.Errorf("Failed to get emails for %s: %v", schedule.AllOnCallGroupHandle, err)
				continue
			}

			err = updateSlackGroup(allOncallEngineerEmails, schedule.AllOnCallGroupHandle)
			if err != nil {
				logrus.Errorf("Failed to update slack group %s: %v", schedule.AllOnCallGroupHandle, err)
				continue
			}
		}
	}

	return nil
}

func appendIfMissing(slice []string, items ...string) []string {
out:
	for _, i := range items {
		for _, ele := range slice {
			if ele == i {
				continue out
			}
		}
		slice = append(slice, i)
	}

	return slice
}

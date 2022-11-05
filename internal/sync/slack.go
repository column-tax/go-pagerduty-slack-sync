package sync

import (
	"fmt"
	"github.com/slack-go/slack"
	"strings"
)

type slackClient struct {
	users      []slack.User
	userGroups []slack.UserGroup
	Client     *slack.Client
}

func newSlackClient(token string) (*slackClient, error) {
	s := slack.New(token)

	userGroups, err := s.GetUserGroups()
	if err != nil {
		return nil, err
	}

	users, err := s.GetUsers()
	if err != nil {
		return nil, err
	}

	return &slackClient{
		users:      users,
		userGroups: userGroups,
		Client:     s,
	}, nil
}

func (s *slackClient) createOrGetUserGroup(handle string) (*slack.UserGroup, error) {
	group := s.findUserGroupByHandle(handle)
	if group != nil {
		return group, nil
	}

	g, err := s.Client.CreateUserGroup(slack.UserGroup{
		Name:   handle,
		Handle: handle,
	})
	if err != nil {
		return nil, err
	}

	return &g, err
}

func (s *slackClient) getSlackIDsFromEmails(emails []string) ([]string, error) {
	var results []string
	for _, email := range emails {
		ID := s.findUserIDByEmail(email)
		if ID == nil {
			return nil, fmt.Errorf("could not find slack user with email: %s", email)
		}
		results = append(results, *ID)
	}
	return results, nil
}

func (s *slackClient) findUserIDByEmail(email string) *string {
	for _, u := range s.users {
		if strings.EqualFold(email, u.Profile.Email) {
			return &u.ID
		}
	}
	return nil
}

func (s *slackClient) findUserGroupByHandle(handle string) *slack.UserGroup {
	for _, g := range s.userGroups {
		if strings.EqualFold(handle, g.Handle) {
			return &g
		}
	}
	return nil
}

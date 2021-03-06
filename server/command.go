package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-zoom/server/zoom"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const COMMAND_HELP = `* |/zoom start| - Start a zoom meeting.`

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "zoom",
		DisplayName:      "Zoom",
		Description:      "Integration with Zoom.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: start",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   text,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) executeCommand(c *plugin.Context, args *model.CommandArgs) (string, error) {
	config := p.getConfiguration()

	split := strings.Fields(args.Command)
	command := split[0]
	action := ""

	if command != "/zoom" {
		return fmt.Sprintf("Command '%s' is not /zoom. Please try again.", command), nil
	}

	if len(split) > 1 {
		action = split[1]
	} else {
		return "Please specify an action for /zoom command.", nil
	}

	userID := args.UserId
	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		return fmt.Sprintf("We could not retrieve user (userId: %v)", args.UserId), nil
	}

	if action == "start" {
		if _, appErr = p.API.GetChannelMember(args.ChannelId, userID); appErr != nil {
			return fmt.Sprintf("We could not get channel members (channelId: %v)", args.ChannelId), nil
		}

		// create a personal zoom meeting
		ru, clientErr := p.zoomClient.GetUser(user.Email)
		if clientErr != nil {
			return "We could not verify your Mattermost account in Zoom. Please ensure that your Mattermost email address matches your Zoom login email address.", nil
		}
		meetingID := ru.Pmi

		zoomURL := strings.TrimSpace(config.ZoomURL)
		if len(zoomURL) == 0 {
			zoomURL = "https://zoom.us"
		}

		meetingURL := fmt.Sprintf("%s/j/%v", zoomURL, meetingID)

		post := &model.Post{
			UserId:    p.botUserID,
			ChannelId: args.ChannelId,
			Message:   fmt.Sprintf("Meeting started at %s.", meetingURL),
			Type:      "custom_zoom",
			Props: map[string]interface{}{
				"meeting_id":       meetingID,
				"meeting_link":     meetingURL,
				"meeting_status":   zoom.WebhookStatusStarted,
				"meeting_personal": true,
			},
		}

		_, appErr := p.API.CreatePost(post)
		if appErr != nil {
			return "Failed to post message. Please try again.", nil
		}
		return "", nil
	}
	return fmt.Sprintf("Unknown action %v", action), nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	msg, err := p.executeCommand(c, args)
	if err != nil {
		p.API.LogWarn("failed to execute command", "error", err.Error())
	}
	if msg != "" {
		p.postCommandResponse(args, msg)
	}
	return &model.CommandResponse{}, nil
}

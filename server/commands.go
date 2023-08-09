package main

import (
	"fmt"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
	"strings"
)

const (
	mailingCommand     = "mailing"
	mailingCommandSend = mailingCommand + " send"
)

const (
	messageFormat = "<@UserName's or ~ChannelName's> <Text>"
)

func (p *Plugin) registerCommands() error {
	commands := [...]model.Command{
		model.Command{
			Trigger:          mailingCommand,
			AutoComplete:     true,
			AutoCompleteDesc: "Отобразить информацию",
		},
		model.Command{
			Trigger:          mailingCommandSend,
			AutoComplete:     true,
			AutoCompleteHint: messageFormat,
			AutoCompleteDesc: "Разослать сообщение",
		},
	}

	for _, command := range commands {
		if err := p.API.RegisterCommand(&command); err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Failed to register %s command", command.Trigger))
		}
	}

	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	trigger := strings.TrimPrefix(args.Command, "/")
	trigger = strings.TrimSuffix(trigger, " ")

	if trigger == mailingCommand {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text: "Этот плагин рассылает заданным пользователям/каналам сообщение. " +
				"Сделайте рассылку с помощью команды: /" + mailingCommandSend + ": " + messageFormat,
		}, nil
	}

	if trigger == mailingCommandSend {
		return p.executeCommandMailing(), nil
	}

	//return an error message when the command has not been detected at all
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf("Unknown command: " + args.Command),
	}, nil
}

func (p *Plugin) executeCommandMailing() *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text: "Этот плагин рассылает заданным пользователям/каналам сообщение. " +
			"Сделайте рассылку с помощью команды: /" + mailingCommandSend + ": " + messageFormat,
	}
}

func (p *Plugin) executeCommandMailingSend(args *model.CommandArgs) *model.CommandResponse {
	givenText := strings.TrimPrefix(args.Command, fmt.Sprintf("/%s", mailingCommandSend))
	givenText = strings.TrimPrefix(givenText, " ")
	fields := strings.Split(givenText, ":")

	if len(fields) < 2 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Пожалуйста, укажите сообщение рассылки в формате: " + messageFormat,
		}
	}

	usersOrChannelsMentions := fields[0]
	userText := fields[1]

	if !strings.ContainsAny(usersOrChannelsMentions, "@") && !strings.ContainsAny(usersOrChannelsMentions, "~") {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Не удалось найти каналы или пользователей =(",
		}
	}

	userOrChannelNames := strings.Split(usersOrChannelsMentions, " ")
	userOrChannelIDs := make([]string, 0)
	excludeUserOrChannelNames := make([]string, 0)

	for _, userOrChannelName := range userOrChannelNames {
		if strings.HasPrefix(userOrChannelName, "@") {
			userName := strings.Replace(userOrChannelName, "@", "", 1)
			if userInfo, _ := p.API.GetUserByUsername(userName); userInfo != nil {
				userOrChannelIDs = append(userOrChannelIDs, userInfo.Id)
			} else {
				excludeUserOrChannelNames = append(excludeUserOrChannelNames, userName)
			}

		} else if strings.HasPrefix(userOrChannelName, "~") {
			channelName := strings.Replace(userOrChannelName, "~", "", 1)
			if channelInfo, _ := p.API.GetChannelByName(args.TeamId, channelName, true); channelInfo != nil {
				userOrChannelIDs = append(userOrChannelIDs, channelInfo.Id)
			} else {
				excludeUserOrChannelNames = append(excludeUserOrChannelNames, channelName)
			}
		}
	}

	for _, channelID := range userOrChannelIDs {
		if _, err := p.API.CreatePost(&model.Post{
			UserId:    args.UserId,
			ChannelId: channelID,
			Message:   userText,
		}); err != nil {
			excludeUserOrChannelNames = append(excludeUserOrChannelNames, channelID)
		}
	}

	if len(excludeUserOrChannelNames) == len(userOrChannelNames) {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Не удалось отправить сообщение =(",
		}
	}

	if len(excludeUserOrChannelNames) != 0 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Сообщение отправлено всем кроме: " + strings.Join(excludeUserOrChannelNames, ", "),
		}
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         "Сообщение отправлено!",
	}
}

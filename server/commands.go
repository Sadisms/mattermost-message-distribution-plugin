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
	messageFormat = "<@UserName's or ~ChannelName's> : <Text>"
)

type userOrChannelInfo struct {
	name   string
	Id     string
	isUser bool
}

func (p *Plugin) registerCommands() error {
	commands := [...]model.Command{
		{
			Trigger:          mailingCommand,
			AutoComplete:     true,
			AutoCompleteDesc: "Отобразить информацию",
		},
		{
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

func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	trigger := strings.TrimPrefix(args.Command, "/")
	trigger = strings.TrimSuffix(trigger, " ")

	if trigger == mailingCommand {
		return p.executeCommandMailing(), nil
	}

	if strings.ContainsAny(trigger, mailingCommandSend) {
		return p.executeCommandMailingSend(args), nil
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

	usersOrChannelsMentions := strings.TrimRight(fields[0], " ")
	userText := strings.TrimLeft(fields[1], " ")

	if !strings.ContainsAny(usersOrChannelsMentions, "@") && !strings.ContainsAny(usersOrChannelsMentions, "~") {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Не удалось найти каналы или пользователей =(",
		}
	}

	if len(userText) == 0 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Не удалось распознать текст =(",
		}
	}

	userOrChannelNames := removeDuplicates(strings.Split(usersOrChannelsMentions, " "))
	userOrChannelIDs := make([]userOrChannelInfo, 0)
	excludeUserOrChannelNames := make([]string, 0)

	for _, userOrChannelName := range userOrChannelNames {
		if strings.HasPrefix(userOrChannelName, "@") {
			userName := strings.Replace(userOrChannelName, "@", "", 1)
			if userInfo, _ := p.API.GetUserByUsername(userName); userInfo != nil {
				userOrChannelIDs = append(userOrChannelIDs, userOrChannelInfo{name: userName, Id: userInfo.Id, isUser: true})
			} else {
				excludeUserOrChannelNames = append(excludeUserOrChannelNames, userName)
			}

		} else if strings.HasPrefix(userOrChannelName, "~") {
			channelName := strings.Replace(userOrChannelName, "~", "", 1)
			if channelInfo, _ := p.API.GetChannelByName(args.TeamId, channelName, true); channelInfo != nil {
				userOrChannelIDs = append(userOrChannelIDs, userOrChannelInfo{name: channelName, Id: channelInfo.Id})
			} else {
				excludeUserOrChannelNames = append(excludeUserOrChannelNames, channelName)
			}
		}
	}

	for _, channelInfo := range userOrChannelIDs {
		var sendMessageFunc func(postModel *model.Post) (*model.Post, *model.AppError)
		if channelInfo.isUser {
			sendMessageFunc = p.createDirectMessage
		} else {
			sendMessageFunc = p.API.CreatePost
		}

		if _, err := sendMessageFunc(&model.Post{
			UserId:    args.UserId,
			ChannelId: channelInfo.Id,
			Message:   userText,
		}); err != nil {
			excludeUserOrChannelNames = append(excludeUserOrChannelNames, channelInfo.name)
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

func (p *Plugin) createDirectMessage(postModel *model.Post) (*model.Post, *model.AppError) {
	direct, err := p.API.GetDirectChannel(postModel.ChannelId, postModel.UserId)
	if err != nil {
		return nil, err
	}

	post, err := p.API.CreatePost(&model.Post{
		UserId:    postModel.UserId,
		ChannelId: direct.Id,
		Message:   postModel.Message,
	})
	if err != nil {
		return nil, err
	}

	return post, nil
}

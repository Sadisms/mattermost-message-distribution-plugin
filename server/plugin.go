package main

import (
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

type Plugin struct {
	plugin.MattermostPlugin
}

func (p *Plugin) OnActivate() error {
	if err := p.registerCommands(); err != nil {
		return errors.Wrap(err, "failed to register commands")
	}

	return nil
}

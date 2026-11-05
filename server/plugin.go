package main

import (
	"strings"
	"sync/atomic"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type configuration struct {
	// Servers is a comma-, semicolon-, space- or newline-separated list of
	// file server host names whose UNC paths are turned into links.
	Servers string
}

// Plugin rewrites posts so that UNC paths on the configured file servers
// become clickable openfolder:// links.
type Plugin struct {
	plugin.MattermostPlugin
	conv atomic.Pointer[Converter]
}

// OnConfigurationChange reloads the server list. With no servers configured
// the plugin leaves every message unchanged.
func (p *Plugin) OnConfigurationChange() error {
	var cfg configuration
	if err := p.API.LoadPluginConfiguration(&cfg); err != nil {
		return err
	}
	p.conv.Store(NewConverter(splitServers(cfg.Servers)))
	return nil
}

func splitServers(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
}

func (p *Plugin) rewrite(post *model.Post) bool {
	if post == nil || post.Type != "" {
		return false // leave system messages alone
	}
	c := p.conv.Load()
	if c == nil {
		return false // no servers configured
	}
	msg := c.Convert(post.Message)
	if msg == post.Message {
		return false
	}
	post.Message = msg
	return true
}

// MessageWillBePosted rewrites the message before it is saved: no bot reply,
// no follow-up edit and no "(edited)" marker.
func (p *Plugin) MessageWillBePosted(_ *plugin.Context, post *model.Post) (*model.Post, string) {
	if p.rewrite(post) {
		return post, ""
	}
	return nil, ""
}

// MessageWillBeUpdated applies the same conversion when a message is edited.
func (p *Plugin) MessageWillBeUpdated(_ *plugin.Context, newPost, _ *model.Post) (*model.Post, string) {
	p.rewrite(newPost)
	return newPost, ""
}

func main() {
	plugin.ClientMain(&Plugin{})
}

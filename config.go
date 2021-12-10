package main

import (
	"fmt"
	"regexp"

	libconfig "github.com/opensourceways/community-robot-lib/config"
)

var (
	checkPRRe          = regexp.MustCompile(`(?mi)^/check-pr\s*$`)
	tideNotification   = "@%s, This pr is not mergeable."
	tideNotificationRe = regexp.MustCompile(fmt.Sprintf(tideNotification, "(.*)"))
)

type configuration struct {
	ConfigItems []botConfig `json:"config_items,omitempty"`
}

func (c *configuration) configFor(org, repo string) *botConfig {
	if c == nil {
		return nil
	}

	items := c.ConfigItems
	v := make([]libconfig.IPluginForRepo, len(items))
	for i := range items {
		v[i] = &items[i]
	}

	if i := libconfig.FindConfig(org, repo, v); i >= 0 {
		return &items[i]
	}
	return nil
}

func (c *configuration) Validate() error {
	if c == nil {
		return nil
	}

	items := c.ConfigItems
	for i := range items {
		if err := items[i].validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *configuration) SetDefault() {
	if c == nil {
		return
	}

	Items := c.ConfigItems
	for i := range Items {
		Items[i].setDefault()
	}
}

type botConfig struct {
	libconfig.PluginForRepo
	Labels        []labelConfig        `json:"labels" required:"true"`
	MissingLabels []missingLabelConfig `json:"missing_labels,omitempty"`
}

func (c *botConfig) setDefault() {
}

func (c *botConfig) validate() error {
	if err := c.PluginForRepo.Validate(); err != nil {
		return err
	}

	if len(c.Labels) == 0 {
		return fmt.Errorf("missing required labels")
	}

	for _, v := range c.Labels {
		if err := v.validate(); err != nil {
			return err
		}
	}

	for _, v := range c.MissingLabels {
		if err := v.validate(); err != nil {
			return err
		}
	}

	return nil
}

type labelConfig struct {
	Label               string `json:"label" required:"true"`
	TipsIfMissing       string `json:"tips_if_missing" required:"true"`
	Person              string `json:"person,omitempty"`
	TipsIfAddedByOthers string `json:"tips_if_added_by_others,omitempty"`
}

type missingLabelConfig struct {
	Label          string `json:"label" required:"true"`
	TipsIfExisting string `json:"tips_if_existing" required:"true"`
}

func (l *labelConfig) validate() error {
	if l.Label == "" {
		return fmt.Errorf("miss label")
	}

	if l.TipsIfMissing == "" {
		return fmt.Errorf("miss TipsIfMissing")
	}

	if l.Person != "" && l.TipsIfAddedByOthers == "" {
		return fmt.Errorf("must set tips_if_added_by_others if person is set")
	}

	return nil
}

func (m *missingLabelConfig) validate() error {
	if m.Label == "" {
		return fmt.Errorf("miss label")
	}

	if m.TipsIfExisting == "" {
		return fmt.Errorf("missing tips_if_existing")
	}

	return nil
}

func (l *labelConfig) isAddByOthers(other string) (bool, string) {
	b := l.Person != "" && l.Person != other

	return b, l.TipsIfAddedByOthers
}

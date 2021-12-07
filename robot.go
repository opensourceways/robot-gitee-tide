package main

import (
	"fmt"

	"github.com/opensourceways/community-robot-lib/giteeclient"
	"github.com/sirupsen/logrus"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	libconfig "github.com/opensourceways/community-robot-lib/config"
	libplugin "github.com/opensourceways/community-robot-lib/giteeplugin"
)

const botName = "tide"

type iClient interface {
	CreatePRComment(owner, repo string, number int32, comment string) error
	DeletePRComment(org, repo string, ID int32) error
	ListPRComments(org, repo string, number int32) ([]sdk.PullRequestComments, error)
	MergePR(owner, repo string, number int32, opt sdk.PullRequestMergePutParam) error
	ListPROperationLogs(org, repo string, number int32) ([]sdk.OperateLog, error)
}

func newRobot(cli iClient) *robot {
	return &robot{cli: cli}
}

type robot struct {
	cli iClient
}

func (bot *robot) NewPluginConfig() libconfig.PluginConfig {
	return &configuration{}
}

func (bot *robot) getConfig(cfg libconfig.PluginConfig, org, repo string) (*botConfig, error) {
	c, ok := cfg.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}

	if bc := c.configFor(org, repo); bc != nil {
		return bc, nil
	}

	return nil, fmt.Errorf("no config for this repo:%s/%s", org, repo)
}

func (bot *robot) RegisterEventHandler(p libplugin.HandlerRegitster) {
	p.RegisterPullRequestHandler(bot.handlePREvent)
	p.RegisterNoteEventHandler(bot.handleNoteEvent)
}

func (bot *robot) handlePREvent(e *sdk.PullRequestEvent, pc libconfig.PluginConfig, log *logrus.Entry) error {
	prInfo := giteeclient.GetPRInfoByPREvent(e)

	if e.GetAction() != "open" {
		log.Debug("Pull request state is not open, skipping...")

		return nil
	}

	if e.GetActionDesc() != giteeclient.PRActionUpdatedLabel {
		return nil
	}

	cfg, err := bot.getConfig(pc, prInfo.Org, prInfo.Repo)
	if err != nil {
		return err
	}

	return bot.handleMerge(prInfo, checkAllLabelsAreReady, cfg)
}

func (bot *robot) handleNoteEvent(e *sdk.NoteEvent, pc libconfig.PluginConfig, log *logrus.Entry) error {
	ne := giteeclient.NewPRNoteEvent(e)
	prInfo := ne.GetPRInfo()

	if !ne.IsCreatingCommentEvent() {
		log.Debug("Event is not a creation of a comment, skipping.")

		return nil
	}

	if !ne.IsPullRequest() || !checkPRRe.MatchString(ne.GetComment()) {
		return nil
	}

	if !ne.PullRequest.Mergeable {

		return bot.writePRComment(prInfo, "Because it conflicts to the target branch.")
	}

	cfg, err := bot.getConfig(pc, prInfo.Org, prInfo.Repo)
	if err != nil {
		return err
	}

	return bot.handleMerge(prInfo, nil, cfg)
}

func (bot *robot) writePRComment(pr giteeclient.PRInfo, comment string) error {
	org := pr.Org
	repo := pr.Repo
	number := pr.Number
	author := pr.Author
	err := bot.deleteOldComments(org, repo, number)

	if err != nil {
		return err
	}

	return bot.cli.CreatePRComment(org, repo, number, fmt.Sprintf(tideNotification, author)+comment)
}

func (bot *robot) deleteOldComments(org, repo string, prNumber int32) error {
	comments, err := bot.cli.ListPRComments(org, repo, prNumber)
	if err != nil {

		return err
	}

	for _, c := range comments {

		if c.User == nil {
			continue
		}

		if tideNotificationRe.MatchString(c.Body) {
			bot.cli.DeletePRComment(org, repo, c.Id)
		}
	}

	return nil
}

func (bot *robot) handleMerge(pr giteeclient.PRInfo, preCheck func(giteeclient.PRInfo, *botConfig) bool, cfg *botConfig) error {
	if preCheck != nil && !preCheck(pr, cfg) {

		return nil
	}

	ops, err := bot.cli.ListPROperationLogs(pr.Org, pr.Repo, pr.Number)
	if err != nil {
		return err
	}

	if noteString := checkPRLabels(pr, cfg, ops); noteString != "" {

		return bot.writePRComment(pr, "\n\n"+noteString)
	}

	return bot.cli.MergePR(pr.Org, pr.Repo, pr.Number, sdk.PullRequestMergePutParam{})
}

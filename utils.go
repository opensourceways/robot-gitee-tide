package main

import (
	"fmt"
	"strings"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/opensourceways/community-robot-lib/giteeclient"
)

func checkPRLabels(pr giteeclient.PRInfo, cfg *botConfig, ops []sdk.OperateLog) string {
	s := checkNeedLabels(pr, cfg, ops)
	s1 := checkMissingLabels(pr, cfg)

	if s != "" && s1 != "" {
		return s + "\n\n" + s1
	}

	return s + s1
}

func checkNeedLabels(pr giteeclient.PRInfo, cfg *botConfig, ops []sdk.OperateLog) string {
	f := func(label labelConfig) string {
		labelName := label.Label

		if !pr.HasLabel(labelName) {

			return label.TipsIfMissing
		}

		log, b := getLatestLog(ops, labelName)
		if !b {

			return fmt.Sprintf("The corresponding operation log is missing." +
				"you should delete the label and add it again by correct way")
		}

		if b, s := label.isAddByOthers(log.who); b {

			return s
		}

		return ""
	}

	v := make([]string, 0, len(cfg.Labels))

	for _, label := range cfg.Labels {

		if s := f(label); s != "" {
			v = append(v, fmt.Sprintf("%s: %s", label.Label, s))
		}
	}

	if n := len(v); n > 0 {
		s := "label is"

		if n > 1 {
			s = "labels are"
		}

		return fmt.Sprintf("**The following %s not ready**.\n\n%s", s, strings.Join(v, "\n\n"))
	}

	return ""
}

func checkMissingLabels(pr giteeclient.PRInfo, cfg *botConfig) string {

	if n := len(cfg.MissingLabels); n == 0 {
		return ""
	}

	v := make([]string, 0, len(cfg.MissingLabels))
	for _, label := range cfg.MissingLabels {

		if pr.HasLabel(label.Label) {
			v = append(v, fmt.Sprintf("%s", label.Label))
		}
	}

	if n := len(v); n > 0 {
		s := "label exists"

		if n > 1 {
			s = "labels exist"
		}

		return fmt.Sprintf("**The following %s**.\n\n%s", s, strings.Join(v, "\n\n"))
	}

	return ""
}

type labelLog struct {
	label string
	who   string
	t     time.Time
}

func getLatestLog(ops []sdk.OperateLog, label string) (labelLog, bool) {
	var t time.Time
	index := -1

	for i := range ops {
		op := &ops[i]

		if !strings.Contains(op.Content, label) {
			continue
		}

		ut, err := time.Parse(time.RFC3339, op.CreatedAt)
		if err != nil {
			continue
		}

		if index < 0 || ut.After(t) {
			t = ut
			index = i
		}
	}

	if index >= 0 {
		user := ops[index].User

		if user != nil && user.Login != "" {

			return labelLog{
				label: label,
				t:     t,
				who:   user.Login,
			}, true
		}
	}

	return labelLog{}, false
}

func checkAllLabelsAreReady(pr giteeclient.PRInfo, cfg *botConfig) bool {
	for _, label := range cfg.Labels {
		if !pr.HasLabel(label.Label) {

			return false
		}
	}

	for _, label := range cfg.MissingLabels {
		if pr.HasLabel(label.Label) {

			return false
		}
	}

	return true
}

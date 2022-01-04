package main

import (
	"time"

	sdk "github.com/opensourceways/go-gitee/gitee"
)

type botComment struct {
	commentID int32
	body      string
	createAt  time.Time
}

func (c botComment) Exists() bool {
	return c.body != ""
}

func findBotComment(
	comments []sdk.PullRequestComments,
	botName string,
	isTargetComment func(string) bool,
) []botComment {
	var bc []botComment
	for i := range comments {
		item := &comments[i]

		if item.User == nil || item.User.Login != botName {
			continue
		}

		if isTargetComment(item.Body) {
			ut, err := time.Parse(time.RFC3339, item.UpdatedAt)
			if err != nil {
				// it is a invalid comment if parsing time failed
				continue
			}

			bc = append(bc, botComment{
				commentID: item.Id,
				body:      item.Body,
				createAt:  ut,
			})
		}
	}

	return bc
}

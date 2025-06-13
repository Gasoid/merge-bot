package handlers

import "regexp"

type Checker struct {
	text      string
	checkFunc func(*Config, *MrInfo) (bool, bool)
}

func checkTitle(mrConfig *Config, info *MrInfo) (bool, bool) {
	match, _ := regexp.MatchString(mrConfig.Rules.TitleRegex, info.Title)
	return match, true
}

func checkDescription(mrConfig *Config, info *MrInfo) (bool, bool) {
	return len(info.Description) > 0, !mrConfig.Rules.AllowEmptyDescription
}

func checkApprovals(mrConfig *Config, info *MrInfo) (bool, bool) {
	return len(info.Approvals) >= mrConfig.Rules.MinApprovals, true
}

func checkApprovers(mrConfig *Config, info *MrInfo) (bool, bool) {
	if len(mrConfig.Rules.Approvers) > 0 {
		for _, a := range mrConfig.Rules.Approvers {
			if _, ok := info.Approvals[a]; !ok {
				return false, true
			}
		}
		return true, true
	}
	return true, false
}

func checkPipelines(mrConfig *Config, info *MrInfo) (bool, bool) {
	return info.FailedPipelines == 0, !mrConfig.Rules.AllowFailingPipelines
}

func checkTests(mrConfig *Config, info *MrInfo) (bool, bool) {
	return info.FailedTests == 0, !mrConfig.Rules.AllowFailingTests
}

var (
	checkers = []Checker{
		{
			text:      "Title meets rules",
			checkFunc: checkTitle,
		},
		{
			text:      "Description meets rules",
			checkFunc: checkDescription,
		},
		{
			text:      "Number of approvals (mr author is ignored)",
			checkFunc: checkApprovals,
		},
		{
			text:      "Required approvers (mr author is ignored)",
			checkFunc: checkApprovers,
		},
		{
			text:      "Pipeline didn't fail ",
			checkFunc: checkPipelines,
		},
		{
			text:      "Tests",
			checkFunc: checkTests,
		},
	}
)

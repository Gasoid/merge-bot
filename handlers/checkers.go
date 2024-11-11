package handlers

import "regexp"

type Checker struct {
	text      string
	checkFunc func(*Config, *MrInfo) (bool, bool)
}

func checkTitle(mrConfig *Config, info *MrInfo) (bool, bool) {
	match, _ := regexp.MatchString(mrConfig.TitleRegex, info.Title)
	return match, true
}

func checkDescription(mrConfig *Config, info *MrInfo) (bool, bool) {
	return len(info.Description) > 0, !mrConfig.AllowEmptyDescription
}

func checkApprovals(mrConfig *Config, info *MrInfo) (bool, bool) {
	return len(info.Approvals) >= mrConfig.MinApprovals, true
}

func checkApprovers(mrConfig *Config, info *MrInfo) (bool, bool) {
	if len(mrConfig.Approvers) > 0 {
		for _, a := range mrConfig.Approvers {
			if _, ok := info.Approvals[a]; !ok {
				return false, true
			}
		}
		return true, true
	}
	return true, false
}

func checkPipelines(mrConfig *Config, info *MrInfo) (bool, bool) {
	return info.FailedPipelines == 0, !mrConfig.AllowFailingPipelines
}

func checkTests(mrConfig *Config, info *MrInfo) (bool, bool) {
	return info.FailedTests == 0, !mrConfig.AllowFailingTests
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
			text:      "Number of approvals",
			checkFunc: checkApprovals,
		},
		{
			text:      "Required approvers",
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

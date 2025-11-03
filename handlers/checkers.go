package handlers

import (
	"fmt"
	"regexp"
)

type CheckResult struct {
	Passed   bool
	Required bool
	Message  string
}

func checkTitle(mrConfig *Config, info *MrInfo) CheckResult {
	if mrConfig.Rules.TitleRegex == "" {
		return CheckResult{Passed: true, Required: false, Message: "No title regex configured"}
	}

	match, err := regexp.MatchString(mrConfig.Rules.TitleRegex, info.Title)
	if err != nil {
		return CheckResult{Passed: false, Required: true, Message: "Invalid title regex pattern"}
	}

	if match {
		return CheckResult{Passed: true, Required: true, Message: "Title matches required pattern"}
	}
	return CheckResult{Passed: false, Required: true, Message: "Title doesn't match required pattern"}
}

func checkDescription(mrConfig *Config, info *MrInfo) CheckResult {
	hasDescription := len(info.Description) > 0
	required := !mrConfig.Rules.AllowEmptyDescription

	if required && !hasDescription {
		return CheckResult{Passed: false, Required: true, Message: "Description is required but empty"}
	}
	if hasDescription {
		return CheckResult{Passed: true, Required: required, Message: "Description provided"}
	}
	return CheckResult{Passed: true, Required: false, Message: "Description not required"}
}

func checkApprovals(mrConfig *Config, info *MrInfo) CheckResult {
	actual := len(info.Approvals)
	required := mrConfig.Rules.MinApprovals

	if actual >= required {
		return CheckResult{
			Passed:   true,
			Required: true,
			Message:  fmt.Sprintf("Has %d approvals (required: %d)", actual, required),
		}
	}
	return CheckResult{
		Passed:   false,
		Required: true,
		Message:  fmt.Sprintf("Has %d approvals, need %d", actual, required),
	}
}

func checkApprovers(mrConfig *Config, info *MrInfo) CheckResult {
	if len(mrConfig.Rules.Approvers) == 0 {
		return CheckResult{Passed: true, Required: false, Message: "No specific approvers required"}
	}

	for _, requiredApprover := range mrConfig.Rules.Approvers {
		if requiredApprover == info.Author {
			return CheckResult{
				Passed:   true,
				Required: true,
				Message:  "Required approver has approved",
			}
		}

		if _, approved := info.Approvals[requiredApprover]; approved {
			return CheckResult{
				Passed:   true,
				Required: true,
				Message:  "Required approver has approved",
			}
		}
	}

	return CheckResult{
		Passed:   false,
		Required: true,
		Message:  "No approvals from required approvers",
	}
}

func checkPipelines(mrConfig *Config, info *MrInfo) CheckResult {
	required := !mrConfig.Rules.AllowFailingPipelines
	passed := info.FailedPipelines == 0

	if passed {
		return CheckResult{Passed: true, Required: required, Message: "All pipelines passed"}
	}

	if required {
		return CheckResult{
			Passed:   false,
			Required: true,
			Message:  fmt.Sprintf("%d pipeline(s) failed", info.FailedPipelines),
		}
	}

	return CheckResult{
		Passed:   true,
		Required: false,
		Message:  fmt.Sprintf("%d pipeline(s) failed (allowed)", info.FailedPipelines),
	}
}

func checkTests(mrConfig *Config, info *MrInfo) CheckResult {
	required := !mrConfig.Rules.AllowFailingTests
	passed := info.FailedTests == 0

	if passed {
		return CheckResult{Passed: true, Required: required, Message: "All tests passed"}
	}

	if required {
		return CheckResult{
			Passed:   false,
			Required: true,
			Message:  fmt.Sprintf("%d test(s) failed", info.FailedTests),
		}
	}

	return CheckResult{
		Passed:   true,
		Required: false,
		Message:  fmt.Sprintf("%d test(s) failed (allowed)", info.FailedTests),
	}
}

var (
	checkers = []func(*Config, *MrInfo) CheckResult{
		checkTitle,
		checkDescription,
		checkApprovals,
		checkApprovers,
		checkPipelines,
		checkTests,
	}
)

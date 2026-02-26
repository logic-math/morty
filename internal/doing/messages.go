// Package doing provides job execution functionality with error handling and retry mechanisms.
package doing

import (
	"fmt"
	"strings"
)

// FriendlyMessage represents a user-friendly error message with guidance.
type FriendlyMessage struct {
	Title           string
	Description     string
	Suggestion      string
	Command         string
	Documentation   string
	Emoji           string
}

// String returns the formatted friendly message.
func (m *FriendlyMessage) String() string {
	var b strings.Builder

	// Title with emoji
	if m.Emoji != "" {
		b.WriteString(fmt.Sprintf("\n%s %s\n", m.Emoji, m.Title))
	} else {
		b.WriteString(fmt.Sprintf("\n%s\n", m.Title))
	}
	b.WriteString(strings.Repeat("=", 50) + "\n")

	// Description
	if m.Description != "" {
		b.WriteString(fmt.Sprintf("\n%s\n", m.Description))
	}

	// Suggestion
	if m.Suggestion != "" {
		b.WriteString(fmt.Sprintf("\n💡 建议: %s\n", m.Suggestion))
	}

	// Command
	if m.Command != "" {
		b.WriteString(fmt.Sprintf("\n🚀 运行: %s\n", m.Command))
	}

	// Documentation
	if m.Documentation != "" {
		b.WriteString(fmt.Sprintf("\n📚 文档: %s\n", m.Documentation))
	}

	return b.String()
}

// GetFriendlyMessage returns a user-friendly message for an error.
// Task 3: Implement friendly error hints
func GetFriendlyMessage(err error) *FriendlyMessage {
	if err == nil {
		return nil
	}

	// Classify the error first
	doingErr := ClassifyError(err)

	switch doingErr.Category {
	case ErrorCategoryPrerequisite:
		return getPrerequisiteMessage(doingErr)
	case ErrorCategoryPlan:
		return getPlanMessage(doingErr)
	case ErrorCategoryExecution:
		return getExecutionMessage(doingErr)
	case ErrorCategoryGit:
		return getGitMessage(doingErr)
	case ErrorCategoryState:
		return getStateMessage(doingErr)
	case ErrorCategoryConfig:
		return getConfigMessage(doingErr)
	case ErrorCategoryTransient:
		return getTransientMessage(doingErr)
	default:
		return getDefaultMessage(doingErr)
	}
}

func getPrerequisiteMessage(err *DoingError) *FriendlyMessage {
	prereqs, _ := err.Context["prerequisites"].([]string)

	msg := &FriendlyMessage{
		Emoji:       "🚫",
		Title:       "前置条件未满足",
		Description: err.Message,
		Suggestion:  "请先完成依赖的 Job 后再执行当前 Job。",
	}

	if len(prereqs) > 0 {
		msg.Description += fmt.Sprintf("\n\n缺少的前置条件:\n")
		for _, p := range prereqs {
			msg.Description += fmt.Sprintf("  - %s\n", p)
		}
	}

	msg.Command = "morty doing"
	return msg
}

func getPlanMessage(err *DoingError) *FriendlyMessage {
	errorType, _ := err.Context["error_type"].(string)

	switch errorType {
	case "plan_not_found":
		return &FriendlyMessage{
			Emoji:         "📋",
			Title:         "计划文件不存在",
			Description:   err.Message + "\n\n在执行 Job 之前，需要先创建计划文件。",
			Suggestion:    "使用 plan 命令创建计划文件。",
			Command:       "morty plan",
			Documentation: "https://morty.dev/docs/plan",
		}

	case "plan_invalid":
		return &FriendlyMessage{
			Emoji:       "⚠️ ",
			Title:       "计划文件格式错误",
			Description: err.Message + "\n\n计划文件的 Markdown 格式可能不正确。",
			Suggestion:  "检查计划文件的语法，确保 Job 和 Task 的定义格式正确。",
			Command:     "morty validate",
		}

	case "job_not_found":
		return &FriendlyMessage{
			Emoji:       "🔍",
			Title:       "Job 不存在",
			Description: err.Message,
			Suggestion:  "请检查 Job 名称是否正确，或查看计划文件中的可用 Jobs。",
			Command:     "morty status",
		}

	default:
		return &FriendlyMessage{
			Emoji:       "⚠️ ",
			Title:       "计划文件错误",
			Description: err.Message,
			Suggestion:  "请检查计划文件是否正确。",
			Command:     "morty plan",
		}
	}
}

func getExecutionMessage(err *DoingError) *FriendlyMessage {
	errorType, _ := err.Context["error_type"].(string)
	retryCount, _ := err.Context["retry_count"].(int)
	maxRetries, _ := err.Context["max_retries"].(int)

	switch errorType {
	case "execution_failed":
		msg := &FriendlyMessage{
			Emoji:       "❌",
			Title:       "执行失败",
			Description: err.Message,
		}

		if retryCount < maxRetries && maxRetries > 0 {
			msg.Suggestion = fmt.Sprintf("已重试 %d/%d 次，正在尝试再次执行...", retryCount, maxRetries)
			msg.Command = "（自动重试中）"
		} else {
			msg.Suggestion = "Job 执行失败，请检查错误日志并修复问题。"
			msg.Command = "morty doing --restart"
		}
		return msg

	case "timeout":
		return &FriendlyMessage{
			Emoji:       "⏱️ ",
			Title:       "执行超时",
			Description: "Job 执行时间超过了设定的超时限制。",
			Suggestion:  "这可能是因为任务过于复杂或系统负载过高。",
			Command:     "morty doing --restart",
		}

	default:
		return &FriendlyMessage{
			Emoji:       "❌",
			Title:       "执行错误",
			Description: err.Message,
			Suggestion:  "请检查错误详情并修复问题。",
			Command:     "morty doing --restart",
		}
	}
}

func getGitMessage(err *DoingError) *FriendlyMessage {
	errorType, _ := err.Context["error_type"].(string)

	switch errorType {
	case "git_not_initialized":
		return &FriendlyMessage{
			Emoji:       "📦",
			Title:       "Git 未初始化",
			Description: "当前目录不是一个 Git 仓库。",
			Suggestion:  "请先初始化 Git 仓库，或使用 --no-git 选项跳过 Git 提交。",
			Command:     "git init",
		}

	case "git_commit_failed":
		return &FriendlyMessage{
			Emoji:       "📝",
			Title:       "Git 提交失败",
			Description: "创建 Git 提交时发生错误。",
			Suggestion:  "请检查 Git 配置和文件状态。",
			Command:     "git status",
		}

	case "git_permission":
		return &FriendlyMessage{
			Emoji:       "🔒",
			Title:       "Git 权限错误",
			Description: "没有权限执行 Git 操作。",
			Suggestion:  "请检查 Git 凭证和仓库权限设置。",
		}

	default:
		return &FriendlyMessage{
			Emoji:       "📦",
			Title:       "Git 错误",
			Description: err.Message,
			Suggestion:  "请检查 Git 配置。",
		}
	}
}

func getStateMessage(err *DoingError) *FriendlyMessage {
	errorType, _ := err.Context["error_type"].(string)

	switch errorType {
	case "state_corrupted":
		recoveryCmd, _ := err.Context["recovery_suggestion"].(string)
		return &FriendlyMessage{
			Emoji:       "💾",
			Title:       "状态文件损坏",
			Description: err.Message,
			Suggestion:  "状态文件可能已损坏，" + recoveryCmd,
			Command:     "rm .morty/status.json && morty doing",
		}

	case "state_not_found":
		return &FriendlyMessage{
			Emoji:       "🆕",
			Title:       "首次运行",
			Description: "未找到状态文件，将创建新的状态。",
			Suggestion:  "这是正常的首次运行行为。",
		}

	default:
		return &FriendlyMessage{
			Emoji:       "💾",
			Title:       "状态错误",
			Description: err.Message,
			Suggestion:  "请检查状态文件。",
		}
	}
}

func getConfigMessage(err *DoingError) *FriendlyMessage {
	return &FriendlyMessage{
		Emoji:       "⚙️ ",
		Title:       "配置错误",
		Description: err.Message,
		Suggestion:  "请检查 morty.yaml 配置文件。",
		Command:     "morty config --validate",
	}
}

func getTransientMessage(err *DoingError) *FriendlyMessage {
	return &FriendlyMessage{
		Emoji:       "🔄",
		Title:       "临时错误",
		Description: err.Message,
		Suggestion:  "这是一个临时性错误，系统将自动重试。",
		Command:     "（自动重试）",
	}
}

func getDefaultMessage(err *DoingError) *FriendlyMessage {
	return &FriendlyMessage{
		Emoji:       "⚠️ ",
		Title:       "发生错误",
		Description: err.Error(),
		Suggestion:  "请检查错误详情并修复问题。",
		Command:     "morty doing --restart",
	}
}

// FormatErrorForDisplay formats an error for display to the user.
func FormatErrorForDisplay(err error) string {
	if err == nil {
		return ""
	}

	msg := GetFriendlyMessage(err)
	if msg == nil {
		return fmt.Sprintf("错误: %v", err)
	}

	return msg.String()
}

// GetQuickFix returns a quick fix command for an error if available.
func GetQuickFix(err error) string {
	if err == nil {
		return ""
	}

	msg := GetFriendlyMessage(err)
	if msg == nil {
		return ""
	}

	return msg.Command
}

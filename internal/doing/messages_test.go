package doing

import (
	"errors"
	"strings"
	"testing"
)

func TestFriendlyMessage_String(t *testing.T) {
	msg := &FriendlyMessage{
		Emoji:         "❌",
		Title:         "Error Occurred",
		Description:   "Something went wrong",
		Suggestion:    "Try again later",
		Command:       "morty doing --restart",
		Documentation: "https://docs.example.com",
	}

	result := msg.String()

	if !strings.Contains(result, "Error Occurred") {
		t.Error("String should contain title")
	}
	if !strings.Contains(result, "Something went wrong") {
		t.Error("String should contain description")
	}
	if !strings.Contains(result, "Try again later") {
		t.Error("String should contain suggestion")
	}
	if !strings.Contains(result, "morty doing --restart") {
		t.Error("String should contain command")
	}
	if !strings.Contains(result, "https://docs.example.com") {
		t.Error("String should contain documentation")
	}
}

func TestFriendlyMessage_String_Minimal(t *testing.T) {
	msg := &FriendlyMessage{
		Title: "Simple Error",
	}

	result := msg.String()

	if !strings.Contains(result, "Simple Error") {
		t.Error("String should contain title")
	}
}

func TestGetFriendlyMessage_Nil(t *testing.T) {
	result := GetFriendlyMessage(nil)
	if result != nil {
		t.Error("Expected nil for nil error")
	}
}

func TestGetFriendlyMessage_Prerequisite(t *testing.T) {
	err := NewDoingError(ErrorCategoryPrerequisite, "前置条件不满足", nil).
		WithContext("prerequisites", []string{"job1", "job2"})

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "前置条件未满足" {
		t.Errorf("Title = %s, want 前置条件未满足", msg.Title)
	}
	if !strings.Contains(msg.Description, "job1") {
		t.Error("Description should contain prerequisites")
	}
}

func TestGetFriendlyMessage_PlanNotFound(t *testing.T) {
	err := NewDoingError(ErrorCategoryPlan, "计划文件不存在", nil).
		WithContext("error_type", "plan_not_found")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "计划文件不存在" {
		t.Errorf("Title = %s, want 计划文件不存在", msg.Title)
	}
	if !strings.Contains(msg.Command, "morty plan") {
		t.Error("Command should suggest 'morty plan'")
	}
}

func TestGetFriendlyMessage_PlanInvalid(t *testing.T) {
	err := NewDoingError(ErrorCategoryPlan, "计划文件格式无效", nil).
		WithContext("error_type", "plan_invalid")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "计划文件格式错误" {
		t.Errorf("Title = %s, want 计划文件格式错误", msg.Title)
	}
}

func TestGetFriendlyMessage_JobNotFound(t *testing.T) {
	err := NewDoingError(ErrorCategoryPlan, "Job 不存在", nil).
		WithContext("error_type", "job_not_found")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "Job 不存在" {
		t.Errorf("Title = %s, want Job 不存在", msg.Title)
	}
}

func TestGetFriendlyMessage_ExecutionFailed(t *testing.T) {
	err := NewDoingError(ErrorCategoryExecution, "执行失败", nil).
		WithContext("error_type", "execution_failed").
		WithContext("retry_count", 2).
		WithContext("max_retries", 3)

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "执行失败" {
		t.Errorf("Title = %s, want 执行失败", msg.Title)
	}
}

func TestGetFriendlyMessage_Timeout(t *testing.T) {
	err := NewDoingError(ErrorCategoryTransient, "执行超时", nil).
		WithContext("error_type", "timeout")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	// Transient errors get a standard title
	if msg.Title != "临时错误" {
		t.Errorf("Title = %s, want 临时错误", msg.Title)
	}
}

func TestGetFriendlyMessage_GitNotInitialized(t *testing.T) {
	err := NewDoingError(ErrorCategoryGit, "Git 未初始化", nil).
		WithContext("error_type", "git_not_initialized")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "Git 未初始化" {
		t.Errorf("Title = %s, want Git 未初始化", msg.Title)
	}
	if !strings.Contains(msg.Command, "git init") {
		t.Error("Command should suggest 'git init'")
	}
}

func TestGetFriendlyMessage_GitCommitFailed(t *testing.T) {
	err := NewDoingError(ErrorCategoryGit, "Git 提交失败", nil).
		WithContext("error_type", "git_commit_failed")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "Git 提交失败" {
		t.Errorf("Title = %s, want Git 提交失败", msg.Title)
	}
}

func TestGetFriendlyMessage_GitPermission(t *testing.T) {
	err := NewDoingError(ErrorCategoryGit, "Git 权限错误", nil).
		WithContext("error_type", "git_permission")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "Git 权限错误" {
		t.Errorf("Title = %s, want Git 权限错误", msg.Title)
	}
}

func TestGetFriendlyMessage_StateCorrupted(t *testing.T) {
	err := NewDoingError(ErrorCategoryState, "状态文件损坏", nil).
		WithContext("error_type", "state_corrupted").
		WithContext("recovery_suggestion", "删除 status.json 后重试")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "状态文件损坏" {
		t.Errorf("Title = %s, want 状态文件损坏", msg.Title)
	}
}

func TestGetFriendlyMessage_StateNotFound(t *testing.T) {
	err := NewDoingError(ErrorCategoryState, "状态文件不存在", nil).
		WithContext("error_type", "state_not_found")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "首次运行" {
		t.Errorf("Title = %s, want 首次运行", msg.Title)
	}
}

func TestGetFriendlyMessage_Config(t *testing.T) {
	err := NewDoingError(ErrorCategoryConfig, "配置错误", nil)

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "配置错误" {
		t.Errorf("Title = %s, want 配置错误", msg.Title)
	}
}

func TestGetFriendlyMessage_Transient(t *testing.T) {
	err := NewDoingError(ErrorCategoryTransient, "临时错误", nil)

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "临时错误" {
		t.Errorf("Title = %s, want 临时错误", msg.Title)
	}
}

func TestGetFriendlyMessage_Unknown(t *testing.T) {
	err := errors.New("unknown error")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "执行错误" {
		t.Errorf("Title = %s, want 执行错误", msg.Title)
	}
}

func TestFormatErrorForDisplay_Nil(t *testing.T) {
	result := FormatErrorForDisplay(nil)
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

func TestFormatErrorForDisplay_WithError(t *testing.T) {
	err := NewDoingError(ErrorCategoryExecution, "test error", nil)
	result := FormatErrorForDisplay(err)

	if result == "" {
		t.Error("Expected non-empty string")
	}
	if !strings.Contains(result, "test error") {
		t.Error("Result should contain error message")
	}
}

func TestGetQuickFix(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "plan not found",
			err:      NewDoingError(ErrorCategoryPlan, "test", nil).WithContext("error_type", "plan_not_found"),
			expected: "morty plan",
		},
		{
			name:     "execution failed",
			err:      NewDoingError(ErrorCategoryExecution, "test", nil).WithContext("error_type", "execution_failed"),
			expected: "morty doing --restart",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetQuickFix(tt.err)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("GetQuickFix() = %s, want to contain %s", result, tt.expected)
			}
		})
	}
}

func TestGetFriendlyMessage_DefaultPlanError(t *testing.T) {
	err := NewDoingError(ErrorCategoryPlan, "generic plan error", nil).
		WithContext("error_type", "unknown_type")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "计划文件错误" {
		t.Errorf("Title = %s, want 计划文件错误", msg.Title)
	}
}

func TestGetFriendlyMessage_DefaultGitError(t *testing.T) {
	err := NewDoingError(ErrorCategoryGit, "generic git error", nil).
		WithContext("error_type", "unknown_git_type")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "Git 错误" {
		t.Errorf("Title = %s, want Git 错误", msg.Title)
	}
}

func TestGetFriendlyMessage_DefaultStateError(t *testing.T) {
	err := NewDoingError(ErrorCategoryState, "generic state error", nil).
		WithContext("error_type", "unknown_state_type")

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Title != "状态错误" {
		t.Errorf("Title = %s, want 状态错误", msg.Title)
	}
}

func TestGetFriendlyMessage_ExecutionFailedMaxRetries(t *testing.T) {
	err := NewDoingError(ErrorCategoryExecution, "执行失败", nil).
		WithContext("error_type", "execution_failed").
		WithContext("retry_count", 5).
		WithContext("max_retries", 3)

	msg := GetFriendlyMessage(err)

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	// When max retries exceeded, should suggest restart
	if !strings.Contains(msg.Command, "restart") {
		t.Error("Command should suggest restart when max retries exceeded")
	}
}

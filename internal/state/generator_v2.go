package state

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/morty/morty/internal/parser/plan"
)

// PlanInfo holds information about a plan file.
type PlanInfo struct {
	Name         string
	DisplayName  string
	FileName     string
	Dependencies []string
	Jobs         []JobInfo
}

// JobInfo holds information about a job.
type JobInfo struct {
	Index         int
	Name          string
	Prerequisites []string
	Tasks         []TaskInfo
}

// TaskInfo holds information about a task.
type TaskInfo struct {
	Index       int
	Description string
	Completed   bool
}

// GenerateStatusV2 generates a V2 status.json from plan files.
func GenerateStatusV2(planDir string) (*StatusV2, error) {
	// Step 1: Scan and parse all plan files
	plans, err := scanPlanFiles(planDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan plan files: %w", err)
	}

	if len(plans) == 0 {
		return nil, fmt.Errorf("no plan files found in %s", planDir)
	}

	// Step 2: Topologically sort modules
	sortedModules, err := topologicalSortModules(plans)
	if err != nil {
		return nil, fmt.Errorf("failed to sort modules: %w", err)
	}

	// Step 3: Build module map for job sorting
	moduleMap := make(map[string]PlanInfo)
	for _, p := range sortedModules {
		moduleMap[p.Name] = p
	}

	// Step 4: Generate module states
	modules := []ModuleStateV2{}
	globalJobIndex := 0
	now := time.Now()

	for moduleIndex, planInfo := range sortedModules {
		// Topologically sort jobs within module
		sortedJobs, err := topologicalSortJobs(planInfo, moduleMap)
		if err != nil {
			return nil, fmt.Errorf("failed to sort jobs in module %s: %w", planInfo.Name, err)
		}

		// Create job states
		jobs := []JobStateV2{}
		for jobIndex, jobInfo := range sortedJobs {
			tasks := []TaskState{}
			for _, taskInfo := range jobInfo.Tasks {
				tasks = append(tasks, TaskState{
					Index:       taskInfo.Index,
					Status:      StatusPending,
					Description: taskInfo.Description,
					UpdatedAt:   now,
				})
			}

			job := JobStateV2{
				Index:          jobIndex,
				GlobalIndex:    globalJobIndex,
				Name:           jobInfo.Name,
				Status:         StatusPending,
				Prerequisites:  jobInfo.Prerequisites,
				TasksTotal:     len(jobInfo.Tasks),
				TasksCompleted: 0,
				LoopCount:      0,
				RetryCount:     0,
				Tasks:          tasks,
				DebugLogs:      []DebugLogEntry{},
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			jobs = append(jobs, job)
			globalJobIndex++
		}

		// Create module state
		module := ModuleStateV2{
			Index:        moduleIndex,
			Name:         planInfo.Name,
			DisplayName:  planInfo.DisplayName,
			PlanFile:     planInfo.FileName,
			Status:       StatusPending,
			Dependencies: planInfo.Dependencies,
			Jobs:         jobs,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		modules = append(modules, module)
	}

	// Step 5: Create status
	status := &StatusV2{
		Version: "2.0",
		Global: GlobalStateV2{
			Status:             StatusPending,
			StartTime:          now,
			LastUpdate:         now,
			CurrentModuleIndex: 0,
			CurrentJobIndex:    0,
			TotalModules:       len(modules),
			TotalJobs:          globalJobIndex,
		},
		Modules: modules,
	}

	return status, nil
}

// scanPlanFiles scans the plan directory and parses all plan files.
func scanPlanFiles(planDir string) ([]PlanInfo, error) {
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return nil, err
	}

	var plans []PlanInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if !strings.HasSuffix(fileName, ".md") || strings.HasPrefix(fileName, "README") {
			continue
		}

		// Read plan file
		planPath := filepath.Join(planDir, fileName)
		content, err := os.ReadFile(planPath)
		if err != nil {
			continue
		}

		// Parse plan
		parsedPlan, err := plan.ParsePlan(string(content))
		if err != nil {
			continue
		}

		// Extract module name from filename (without .md)
		moduleName := strings.TrimSuffix(fileName, ".md")

		// Extract jobs
		jobs := []JobInfo{}
		for _, job := range parsedPlan.Jobs {
			tasks := []TaskInfo{}
			for _, task := range job.Tasks {
				tasks = append(tasks, TaskInfo{
					Index:       task.Index,
					Description: task.Description,
					Completed:   task.Completed,
				})
			}

			jobs = append(jobs, JobInfo{
				Index:         job.Index,
				Name:          job.Name,
				Prerequisites: job.Prerequisites,
				Tasks:         tasks,
			})
		}

		// Filter dependencies (remove "无")
		deps := []string{}
		for _, dep := range parsedPlan.Dependencies {
			if dep != "无" && dep != "" {
				deps = append(deps, dep)
			}
		}

		planInfo := PlanInfo{
			Name:         moduleName,
			DisplayName:  parsedPlan.Name,
			FileName:     fileName,
			Dependencies: deps,
			Jobs:         jobs,
		}
		plans = append(plans, planInfo)
	}

	return plans, nil
}

// topologicalSortModules performs topological sort on modules using Kahn's algorithm.
func topologicalSortModules(plans []PlanInfo) ([]PlanInfo, error) {
	// Build dependency map
	deps := make(map[string][]string)
	allModules := make(map[string]PlanInfo)

	for _, p := range plans {
		allModules[p.Name] = p
		deps[p.Name] = p.Dependencies
	}

	// Expand __ALL__ dependencies
	for name, d := range deps {
		if len(d) == 1 && d[0] == "__ALL__" {
			expanded := []string{}
			for otherName := range allModules {
				if otherName != name {
					expanded = append(expanded, otherName)
				}
			}
			deps[name] = expanded
		}
	}

	// Calculate in-degrees
	inDegree := make(map[string]int)
	for name := range allModules {
		inDegree[name] = len(deps[name])
	}

	// Initialize queue with modules that have no dependencies
	queue := []string{}
	for name := range allModules {
		if inDegree[name] == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // Stable ordering

	// Kahn's algorithm
	result := []PlanInfo{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		result = append(result, allModules[current])

		// Update in-degrees
		for name, d := range deps {
			for _, dep := range d {
				if dep == current {
					inDegree[name]--
					if inDegree[name] == 0 {
						queue = append(queue, name)
						sort.Strings(queue)
					}
				}
			}
		}
	}

	// Check for cycles
	if len(result) != len(plans) {
		return nil, fmt.Errorf("cycle detected in module dependencies")
	}

	return result, nil
}

// topologicalSortJobs performs topological sort on jobs within a module.
func topologicalSortJobs(module PlanInfo, allModules map[string]PlanInfo) ([]JobInfo, error) {
	if len(module.Jobs) == 0 {
		return []JobInfo{}, nil
	}

	// Build dependency map (job index -> prerequisite job indices)
	deps := make(map[int][]int)
	jobByIndex := make(map[int]JobInfo)

	for _, job := range module.Jobs {
		jobByIndex[job.Index] = job
		prereqIndices := []int{}

		for _, prereq := range job.Prerequisites {
			prereq = strings.TrimSpace(prereq)

			// Parse job_N format (same module)
			jobRefPattern := regexp.MustCompile(`^job_(\d+)(?:\s*-\s*.*)?$`)
			if matches := jobRefPattern.FindStringSubmatch(prereq); matches != nil {
				var idx int
				fmt.Sscanf(matches[1], "%d", &idx)
				prereqIndices = append(prereqIndices, idx)
				continue
			}

			// Parse module:job_N format (cross-module)
			// Cross-module dependencies are already handled by module ordering
			// We don't need to include them in the job-level dependency graph
		}

		deps[job.Index] = prereqIndices
	}

	// Calculate in-degrees
	inDegree := make(map[int]int)
	for _, job := range module.Jobs {
		inDegree[job.Index] = len(deps[job.Index])
	}

	// Initialize queue
	queue := []int{}
	for _, job := range module.Jobs {
		if inDegree[job.Index] == 0 {
			queue = append(queue, job.Index)
		}
	}
	sort.Ints(queue)

	// Kahn's algorithm
	result := []JobInfo{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		result = append(result, jobByIndex[current])

		// Update in-degrees
		for idx, d := range deps {
			for _, dep := range d {
				if dep == current {
					inDegree[idx]--
					if inDegree[idx] == 0 {
						queue = append(queue, idx)
						sort.Ints(queue)
					}
				}
			}
		}
	}

	// Check for cycles
	if len(result) != len(module.Jobs) {
		return nil, fmt.Errorf("cycle detected in job dependencies")
	}

	return result, nil
}

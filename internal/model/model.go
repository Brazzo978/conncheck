package model

import "time"

const Version = "0.1.0"

type Result struct {
	Version     string       `json:"version" xml:"version"`
	StartedAt   time.Time    `json:"started_at" xml:"started_at"`
	FinishedAt  time.Time    `json:"finished_at" xml:"finished_at"`
	Summary     Summary      `json:"summary" xml:"summary"`
	Findings    []Finding    `json:"findings" xml:"findings>finding"`
	Tests       []TestResult `json:"tests" xml:"tests>test"`
	Environment Environment  `json:"environment" xml:"environment"`
}

type Summary struct {
	StatusCounts map[string]int `json:"status_counts" xml:"status_counts"`
}

type Environment struct {
	OS       string `json:"os" xml:"os"`
	Arch     string `json:"arch" xml:"arch"`
	Hostname string `json:"hostname" xml:"hostname"`
	Timezone string `json:"timezone" xml:"timezone"`
}

type TestResult struct {
	Name      string            `json:"name" xml:"name"`
	Status    string            `json:"status" xml:"status"`
	Metrics   map[string]string `json:"metrics" xml:"metrics>metric"`
	Findings  []Finding         `json:"findings" xml:"findings>finding"`
	Evidence  []Evidence        `json:"evidence" xml:"evidence>item"`
	StartedAt time.Time         `json:"started_at" xml:"started_at"`
	EndedAt   time.Time         `json:"ended_at" xml:"ended_at"`
}

type Evidence struct {
	Label string `json:"label" xml:"label"`
	Path  string `json:"path" xml:"path"`
	Note  string `json:"note,omitempty" xml:"note,omitempty"`
}

type Finding struct {
	Severity string `json:"severity" xml:"severity"`
	Title    string `json:"title" xml:"title"`
	Detail   string `json:"detail" xml:"detail"`
}

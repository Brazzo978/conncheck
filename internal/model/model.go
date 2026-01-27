package model

import (
	"encoding/xml"
	"sort"
	"time"
)

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
	StatusCounts IntMap `json:"status_counts" xml:"status_counts"`
}

type Environment struct {
	OS       string `json:"os" xml:"os"`
	Arch     string `json:"arch" xml:"arch"`
	Hostname string `json:"hostname" xml:"hostname"`
	Timezone string `json:"timezone" xml:"timezone"`
}

type TestResult struct {
	Name      string     `json:"name" xml:"name"`
	Status    string     `json:"status" xml:"status"`
	Metrics   StringMap  `json:"metrics" xml:"metrics"`
	Findings  []Finding  `json:"findings" xml:"findings>finding"`
	Evidence  []Evidence `json:"evidence" xml:"evidence>item"`
	StartedAt time.Time  `json:"started_at" xml:"started_at"`
	EndedAt   time.Time  `json:"ended_at" xml:"ended_at"`
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

type StringMap map[string]string

type IntMap map[string]int

func (m StringMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return marshalMapXML(e, start, "metric", stringMapItems(m))
}

func (m IntMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return marshalMapXML(e, start, "item", intMapItems(m))
}

type mapItem struct {
	key   string
	value any
}

func stringMapItems(m StringMap) []mapItem {
	items := make([]mapItem, 0, len(m))
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		items = append(items, mapItem{key: key, value: m[key]})
	}
	return items
}

func intMapItems(m IntMap) []mapItem {
	items := make([]mapItem, 0, len(m))
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		items = append(items, mapItem{key: key, value: m[key]})
	}
	return items
}

func marshalMapXML(e *xml.Encoder, start xml.StartElement, itemName string, items []mapItem) error {
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, item := range items {
		itemStart := xml.StartElement{Name: xml.Name{Local: itemName}}
		if err := e.EncodeToken(itemStart); err != nil {
			return err
		}
		if err := e.EncodeElement(item.key, xml.StartElement{Name: xml.Name{Local: "key"}}); err != nil {
			return err
		}
		if err := e.EncodeElement(item.value, xml.StartElement{Name: xml.Name{Local: "value"}}); err != nil {
			return err
		}
		if err := e.EncodeToken(itemStart.End()); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
}

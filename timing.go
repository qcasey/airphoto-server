package main

// TODO
import "time"

// DebugTiming holds timing information for a task, as well as its subtasks
type DebugTiming struct {
	Name        string
	TimeElapsed time.Time
	Subtimings  []DebugTiming
}

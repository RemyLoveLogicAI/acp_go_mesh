// Package types defines the shared A2A-compatible wire types used by the
// harness, manager, and worker. All new fields are optional so agents that
// don't yet emit them continue to interoperate.
//
// NOTE: This package is kept for backward compatibility. New code should
// import "acp-mesh/pkg/a2a" directly. The types here are re-exports.
package types

import (
	"acp-mesh/pkg/a2a"
)

// Re-export A2A types for backward compatibility.
// These aliases allow existing code to use types.XXX while migrating.

type (
	AgentCard   = a2a.AgentCard
	AgentSkill  = a2a.AgentSkill
	TaskState   = a2a.TaskState
	Task        = a2a.Task
	TaskStatus  = a2a.TaskStatus
	A2AEnvelope = a2a.A2AEnvelope
	TaskUpdate  = a2a.TaskUpdate
	Session     = a2a.Session
	Message     = a2a.Message
	Part        = a2a.Part
	Artifact    = a2a.Artifact
)

// Re-export task state constants for backward compatibility.
const (
	TaskSubmitted     = a2a.TaskStateSubmitted
	TaskWorking       = a2a.TaskStateWorking
	TaskInputRequired = a2a.TaskStateInputRequired
	TaskCompleted     = a2a.TaskStateCompleted
	TaskFailed        = a2a.TaskStateFailed
	TaskCanceled      = a2a.TaskStateCanceled
)

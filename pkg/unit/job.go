package unit

import "github.com/unng-lab/endless/pkg/geom"

// JobStatus describes the result a unit reports back to the actor that issued a job.
// The status stays deliberately small because the current stress harness only needs to
// distinguish successful route completion from any kind of cancellation or rejection.
type JobStatus uint8

const (
	JobStatusCompleted JobStatus = iota
	JobStatusFailed
)

func (s JobStatus) String() string {
	switch s {
	case JobStatusCompleted:
		return "completed"
	case JobStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// MoveJob is the job payload the actor sends to a mobile unit. The job keeps both the actor
// identity and the target tile so the resulting report can be matched without external state.
type MoveJob struct {
	ID          int64
	ActorID     int64
	UnitID      int64
	TargetTileX int
	TargetTileY int
}

// JobReport is the event a unit emits once a job either reaches its destination or becomes
// invalid before completion.
type JobReport struct {
	JobID       int64
	ActorID     int64
	UnitID      int64
	Status      JobStatus
	TargetTileX int
	TargetTileY int
}

type moveJobState struct {
	job      MoveJob
	assigned bool
}

type jobReportingUnit interface {
	drainJobReports() []JobReport
}

// AssignMoveJob installs a new actor-issued move order onto the unit. Any previously tracked
// job is reported as failed first because the actor must be told that its earlier request no
// longer owns the movement state.
func (u *NonStaticUnit) AssignMoveJob(job MoveJob, path []geom.Point) {
	u.failAssignedMoveJob()
	u.moveJob = moveJobState{
		job:      job,
		assigned: true,
	}
	u.setPathWithoutJobCancel(path)
	u.completeAssignedMoveJobIfFinished()
}

// drainJobReports hands the manager a snapshot of all statuses the unit has emitted since the
// last drain. The defensive copy keeps worker updates and manager-side aggregation isolated.
func (u *NonStaticUnit) drainJobReports() []JobReport {
	if len(u.jobReports) == 0 {
		return nil
	}

	reports := append([]JobReport(nil), u.jobReports...)
	u.jobReports = u.jobReports[:0]
	return reports
}

// completeAssignedMoveJobIfFinished closes the current move job once the unit has become
// fully idle again. Waiting for the idle state instead of only checking the path slice ensures
// the actor hears about completion after the last interpolated segment has logically ended.
func (u *NonStaticUnit) completeAssignedMoveJobIfFinished() {
	if !u.moveJob.assigned || u.Base().IsMoving() {
		return
	}

	u.emitMoveJobReport(JobStatusCompleted, u.moveJob.job)
	u.moveJob = moveJobState{}
}

// failAssignedMoveJob marks the current actor-issued move as unfinished. This is used when a
// manual command overrides the route, the unit dies or some other state reset discards the
// actor's ownership of the movement command.
func (u *NonStaticUnit) failAssignedMoveJob() {
	if !u.moveJob.assigned {
		return
	}

	u.emitMoveJobReport(JobStatusFailed, u.moveJob.job)
	u.moveJob = moveJobState{}
}

func (u *NonStaticUnit) emitMoveJobReport(status JobStatus, job MoveJob) {
	u.jobReports = append(u.jobReports, JobReport{
		JobID:       job.ID,
		ActorID:     job.ActorID,
		UnitID:      job.UnitID,
		Status:      status,
		TargetTileX: job.TargetTileX,
		TargetTileY: job.TargetTileY,
	})
}

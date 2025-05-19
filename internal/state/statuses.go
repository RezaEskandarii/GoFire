package state

type JobStatus string

const (
	StatusQueued     JobStatus = "queued"
	StatusProcessing JobStatus = "processing"
	StatusSucceeded  JobStatus = "succeeded"
	StatusFailed     JobStatus = "failed"
	StatusRetrying   JobStatus = "retrying"
	StatusDead       JobStatus = "dead"
)

func (s JobStatus) String() string {
	return string(s)
}

var AllStatuses = []JobStatus{
	StatusQueued,
	StatusProcessing,
	StatusSucceeded,
	StatusFailed,
	StatusRetrying,
	StatusDead,
}

type Transition struct {
	From JobStatus
	To   JobStatus
}

var ValidTransitions = []Transition{
	{From: StatusQueued, To: StatusProcessing},
	{From: StatusProcessing, To: StatusSucceeded},
	{From: StatusProcessing, To: StatusFailed},
	{From: StatusFailed, To: StatusRetrying},
	{From: StatusRetrying, To: StatusProcessing},
	{From: StatusFailed, To: StatusDead},
}

func IsValidTransition(from, to JobStatus) bool {
	for _, t := range ValidTransitions {
		if t.From == from && t.To == to {
			return true
		}
	}
	return false
}

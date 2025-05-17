package task

type JobStatus string

const (
	StatusQueued     JobStatus = "queued"
	StatusProcessing JobStatus = "processing"
	StatusSucceeded  JobStatus = "succeeded"
	StatusFailed     JobStatus = "failed"
	StatusRetrying   JobStatus = "retrying"
	StatusCancelled  JobStatus = "cancelled"
	StatusExpired    JobStatus = "expired"
	StatusDead       JobStatus = "dead"
)

type StateTransition struct {
	From JobStatus
	To   JobStatus
}

var ValidTransitions = []StateTransition{
	{From: StatusQueued, To: StatusProcessing},
	{From: StatusProcessing, To: StatusSucceeded},
	{From: StatusProcessing, To: StatusFailed},
	{From: StatusFailed, To: StatusRetrying},
	{From: StatusRetrying, To: StatusProcessing},
	{From: StatusFailed, To: StatusDead},
	{From: StatusQueued, To: StatusCancelled},
}

func IsValidTransition(from, to JobStatus) bool {
	for _, t := range ValidTransitions {
		if t.From == from && t.To == to {
			return true
		}
	}
	return false
}

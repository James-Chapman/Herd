package common

type Status int

const (
	WAITING   Status = iota // 0
	STARTING                // 1
	RUNNING                 // 2
	SUCCESS                 // 3
	FAILED                  // 4
	CANCELLED               // 5
)

type Job struct {
	Command string
	Args    []string
	Status  Status
}

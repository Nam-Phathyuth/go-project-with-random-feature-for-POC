package pkg

type TaskStatus string

const (
	TODO      TaskStatus = "TODO"
	COMPLETED TaskStatus = "COMPLETED"
	PENDING   TaskStatus = "PENDING"
)

func (s TaskStatus) IsValid() bool {
	switch s {
	case TODO, PENDING, COMPLETED:
		return true
	}
	return false
}

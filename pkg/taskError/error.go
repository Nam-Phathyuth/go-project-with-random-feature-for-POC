package taskError

import (
	"fmt"
)

type ErrorNotFound struct {
	err error
}

func (e *ErrorNotFound) Error() string {
	return fmt.Sprintf("NotFound!!")
}

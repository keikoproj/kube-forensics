package utils

import "fmt"

// This struct used to report errors.
type CommandError struct {
	ID     int
	Result string
}

func (c CommandError) Error() string {
	return fmt.Sprintf("id = %d; result = %s", c.ID, c.Result)
}

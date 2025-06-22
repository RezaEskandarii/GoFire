package custom_errors

import (
	"errors"
	"fmt"
)

type ValidationError struct {
	Errors []error `json:"errors"`
}

func (c *ValidationError) Add(err error) {
	c.Errors = append(c.Errors, err)
}

func (c *ValidationError) HasError() bool {
	return len(c.Errors) > 0
}

func (c *ValidationError) Error() string {
	if len(c.Errors) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", errors.Join(c.Errors...))
}

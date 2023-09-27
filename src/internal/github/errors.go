package github

import "fmt"

type ProviderError struct {
	Message string
	Code    int
}

func (p ProviderError) Error() string {
	return fmt.Sprintf("%d, %s", p.Code, p.Message)
}

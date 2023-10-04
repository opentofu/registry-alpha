package providers

import "fmt"

type FetchErrorCode int

const (
	ErrCodeReleaseNotFound       FetchErrorCode = 1
	ErrCodeAssetNotFound         FetchErrorCode = 2
	ErrCodeSHASumsNotFound       FetchErrorCode = 3
	ErrCodeManifestNotFound      FetchErrorCode = 4
	ErrCodeCouldNotGetPublicKeys FetchErrorCode = 5
)

type FetchError struct {
	Inner   error
	Message string
	Code    FetchErrorCode
}

func (p *FetchError) Error() string {
	return fmt.Sprintf("%d, %s", p.Code, p.Message)
}

func (p *FetchError) Unwrap() error {
	return p.Inner
}

func newFetchError(message string, code FetchErrorCode, err error) error {
	return &FetchError{
		Message: message,
		Code:    code,
		Inner:   err,
	}
}

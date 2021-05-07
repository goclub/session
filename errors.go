package sess

import "errors"

func AsEmptySessionID(err error) (emptySessionID *ErrEmptySessionID, as bool) {
	as = errors.As(err, &emptySessionID)
	return
}
type ErrEmptySessionID struct{
	StoreKey string
}
func (e *ErrEmptySessionID) Error() string {
	return "goclub/session: sessionID can not be empty string"
}

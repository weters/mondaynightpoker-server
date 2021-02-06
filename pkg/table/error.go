package table

// UserError is an error that is safe to return in a response
type UserError string

func (u UserError) Error() string {
	return string(u)
}

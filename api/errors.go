package api

type NotImplemented struct{}

func (e NotImplemented) Error() string {
	return "not implemented"
}

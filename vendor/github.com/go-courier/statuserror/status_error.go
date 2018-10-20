package statuserror

type StatusError interface {
	Key() string
	Code() int
	Msg() string
	CanBeTalkError() bool
	StatusCode() int
	StatusErr() *StatusErr
	Error() string
}

type StatusErrorWithServiceCode interface {
	ServiceCode() int
}

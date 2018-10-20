package statuserror

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func FromErr(err error) *StatusErr {
	if err == nil {
		return nil
	}

	if statusErrCode, ok := err.(StatusError); ok {
		return statusErrCode.StatusErr()
	}

	if statusErr, ok := err.(*StatusErr); ok {
		return statusErr
	}

	return NewUnknownErr().WithDesc(err.Error())
}

func NewUnknownErr() *StatusErr {
	return NewStatusErr("UnknownError", http.StatusInternalServerError*1e6, "unknown error")
}

func NewStatusErr(key string, code int, msg string) *StatusErr {
	return &StatusErr{
		Key:  key,
		Code: code,
		Msg:  msg,
	}
}

type StatusErr struct {
	// key of err
	Key string `json:"key" xml:"key"`
	// unique err code
	Code int `json:"code" xml:"code"`
	// msg of err
	Msg string `json:"msg" xml:"msg"`
	// desc of err
	Desc string `json:"desc" xml:"desc"`
	// can be task error
	// for client to should error msg to end user
	CanBeTalkError bool `json:"canBeTalkError" xml:"canBeTalkError"`

	// request ID or other request context
	ID string `json:"id" xml:"id"`
	// error tracing
	Sources []string `json:"sources" xml:"sources"`
	// error in where fields
	ErrorFields ErrorFields `json:"errorFields" xml:"errorFields"`
}

func ParseStatusErrSummary(s string) (*StatusErr, error) {
	if !reStatusErrSummary.Match([]byte(s)) {
		return nil, fmt.Errorf("unsupported status err summary: %s", s)
	}

	matched := reStatusErrSummary.FindStringSubmatch(s)

	code, _ := strconv.ParseInt(matched[2], 10, 64)

	return &StatusErr{
		Key:            matched[1],
		Code:           int(code),
		Msg:            matched[3],
		CanBeTalkError: matched[4] != "",
	}, nil
}

// @err[UnknownError][500000000][unknown error]
var reStatusErrSummary = regexp.MustCompile(`@StatusErr\[(.+)\]\[(.+)\]\[(.+)\](!)?`)

func (statusErr *StatusErr) Summary() string {
	s := fmt.Sprintf(
		`@StatusErr[%s][%d][%s]`,
		statusErr.Key,
		statusErr.Code,
		statusErr.Msg,
	)

	if statusErr.CanBeTalkError {
		return s + "!"
	}
	return s
}

func (statusErr *StatusErr) Is(err error) bool {
	return FromErr(err).Code == statusErr.Code
}

func StatusCodeFromCode(code int) int {
	strCode := fmt.Sprintf("%d", code)
	if len(strCode) < 3 {
		return 0
	}
	statusCode, _ := strconv.Atoi(strCode[:3])
	return statusCode
}

func (statusErr *StatusErr) StatusCode() int {
	return StatusCodeFromCode(statusErr.Code)
}

func (statusErr *StatusErr) Error() string {
	s := fmt.Sprintf(
		"[%s]%s%s",
		strings.Join(statusErr.Sources, ","),
		statusErr.Summary(),
		statusErr.ErrorFields,
	)

	if statusErr.Desc != "" {
		s += " " + statusErr.Desc
	}

	return s
}

func (statusErr StatusErr) WithMsg(msg string) *StatusErr {
	statusErr.Msg = msg
	return &statusErr
}

func (statusErr StatusErr) WithDesc(desc string) *StatusErr {
	statusErr.Desc = desc
	return &statusErr
}

func (statusErr StatusErr) WithID(id string) *StatusErr {
	statusErr.ID = id
	return &statusErr
}

func (statusErr StatusErr) AppendSource(sourceName string) *StatusErr {
	length := len(statusErr.Sources)
	if length == 0 || statusErr.Sources[length-1] != sourceName {
		statusErr.Sources = append(statusErr.Sources, sourceName)
	}
	return &statusErr
}

func (statusErr StatusErr) EnableErrTalk() *StatusErr {
	statusErr.CanBeTalkError = true
	return &statusErr
}

func (statusErr StatusErr) DisableErrTalk() *StatusErr {
	statusErr.CanBeTalkError = false
	return &statusErr
}

func (statusErr StatusErr) AppendErrorField(in string, field string, msg string) *StatusErr {
	statusErr.ErrorFields = append(statusErr.ErrorFields, NewErrorField(in, field, msg))
	return &statusErr
}

func (statusErr StatusErr) AppendErrorFields(errorFields ...*ErrorField) *StatusErr {
	statusErr.ErrorFields = append(statusErr.ErrorFields, errorFields...)
	return &statusErr
}

func NewErrorField(in string, field string, msg string) *ErrorField {
	return &ErrorField{
		In:    in,
		Field: field,
		Msg:   msg,
	}
}

type ErrorField struct {
	// field path
	// prop.slice[2].a
	Field string `json:"field" xml:"field"`
	// msg
	Msg string `json:"msg" xml:"msg"`
	// location
	// eq. body, query, header, path, formData
	In string `json:"in" xml:"in"`
}

func (s ErrorField) String() string {
	return s.Field + " in " + s.In + " - " + s.Msg
}

type ErrorFields []*ErrorField

func (fields ErrorFields) String() string {
	if len(fields) == 0 {
		return ""
	}

	sort.Sort(fields)

	buf := &bytes.Buffer{}
	buf.WriteString("<")
	for i, f := range fields {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(f.String())
	}
	buf.WriteString(">")
	return buf.String()
}

func (fields ErrorFields) Len() int {
	return len(fields)
}

func (fields ErrorFields) Swap(i, j int) {
	fields[i], fields[j] = fields[j], fields[i]
}

func (fields ErrorFields) Less(i, j int) bool {
	return fields[i].Field < fields[j].Field
}

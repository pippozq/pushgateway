package envconf

import (
	"bytes"
	"fmt"
	"strings"
)

func NewPathWalker() *PathWalker {
	return &PathWalker{
		path: []interface{}{},
	}
}

type PathWalker struct {
	prefix string
	path   []interface{}
}

func (pw *PathWalker) Enter(i interface{}) {
	pw.path = append(pw.path, i)
}

func (pw *PathWalker) Exit() {
	pw.path = pw.path[:len(pw.path)-1]
}

func (pw *PathWalker) Paths() []interface{} {
	return pw.path
}

func (pw *PathWalker) String() string {
	buf := bytes.NewBuffer(nil)

	path := pw.path

	for i, key := range path {
		if i > 0 {
			buf.WriteRune('_')
		}
		switch v := key.(type) {
		case string:
			buf.WriteString(v)
		case int:
			buf.WriteString(fmt.Sprintf("%d", v))
		}
	}
	return strings.ToUpper(buf.String())
}

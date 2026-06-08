package common

import (
	"errors"
	"fmt"
	"strings"

	"github.com/deposist/s-ui-x/logger"
)

func NewErrorf(format string, a ...interface{}) error {
	msg := fmt.Sprintf(format, a...)
	return errors.New(msg)
}

func NewError(a ...interface{}) error {
	var builder strings.Builder
	for i, item := range a {
		if i > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(fmt.Sprint(item))
	}
	msg := builder.String()
	return errors.New(msg)
}

func Recover(msg string) interface{} {
	panicErr := recover()
	if panicErr != nil {
		if msg != "" {
			logger.Error(msg, "panic:", panicErr)
		}
	}
	return panicErr
}

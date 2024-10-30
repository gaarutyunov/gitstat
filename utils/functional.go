package utils

import (
	"context"
	"github.com/sirupsen/logrus"
)

func Must[T any](res T, err error) T {
	if err != nil {
		logrus.Fatal(err)
	}
	return res
}

func Ignore[T any](res T, err error) T {
	if err != nil {
		logrus.Error(err)
	}
	return res
}

func WithContext(ctx context.Context, f func(ctx context.Context)) func() {
	return func() {
		f(ctx)
	}
}

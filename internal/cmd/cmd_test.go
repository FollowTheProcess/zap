package cmd_test

import (
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/cmd"
)

func TestSmoke(t *testing.T) {
	_, err := cmd.Build()
	test.Ok(t, err)
}

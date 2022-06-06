package main

import (
	"bytes"
	"github.com/sethvargo/go-githubactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewFromInputs(t *testing.T) {
	actionLog := bytes.NewBuffer(nil)
	envMap := map[string]string{
		"INPUT_REF":    "feature/VTHP-100-test",
		"INPUT_PREFIX": "VTHP",
	}
	getenv := func(key string) string {
		return envMap[key]
	}
	action := githubactions.New(
		githubactions.WithWriter(actionLog),
		githubactions.WithGetenv(getenv),
	)
	err := run(action)
	assert.Nil(t, err)
	assert.Equal(t, "::set-output name=ticket::VTHP-100\n", actionLog.String())
}

//go:build ignore

package mcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandsSuggest(t *testing.T) {
	resetDefaultApp()
	addTestGithubCliCommands()

	suggest := defaultApp.cmds.suggest("aapi")
	assert.True(t, len(suggest) > 0)
	assert.Equal(t, "api", suggest[0])

	suggest = defaultApp.cmds.suggest("extenson create")
	assert.True(t, len(suggest) > 0)
	assert.Equal(t, "extension create", suggest[0])

	suggest = defaultApp.cmds.suggest("extension crate")
	assert.True(t, len(suggest) > 0)
	assert.Equal(t, "extension create", suggest[0])
}

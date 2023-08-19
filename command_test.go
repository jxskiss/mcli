package mcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCommandsGrouping(t *testing.T) {
	cmds := commands{
		{
			Name:  "cmd0",
			level: 1,
		},
		{
			Name:    "cmd1",
			Hidden:  false,
			cmdOpts: []CmdOpt{WithCategory("group1")},
			level:   1,
		},
		{
			Name:    "cmd1 sub1",
			Hidden:  false,
			cmdOpts: []CmdOpt{WithCategory("group1")},
			level:   2,
		},
		{
			Name:    "cmd2",
			Hidden:  true,
			cmdOpts: []CmdOpt{WithCategory("group1")},
			level:   1,
		},
		{
			Name:    "cmd3",
			Hidden:  true,
			cmdOpts: []CmdOpt{WithCategory("group3")},
			level:   1,
		},
		{
			Name:   "cmd4",
			Hidden: false,
			level:  1,
		},
		{
			Name:    "cmd5",
			Hidden:  false,
			cmdOpts: []CmdOpt{WithCategory("group5")},
			level:   1,
		},
	}
	cmdGroups, hasCategoreis := cmds.groupByCategory()
	require.True(t, hasCategoreis)
	t.Log(cmdGroups)
	require.Len(t, cmdGroups, 4)

	assert.Equal(t, "group1", cmdGroups[0].category)
	assert.Len(t, cmdGroups[0].commands, 2)

	assert.Equal(t, "group3", cmdGroups[1].category)
	assert.Len(t, cmdGroups[1].commands, 1)

	assert.Equal(t, "group5", cmdGroups[2].category)
	assert.Len(t, cmdGroups[2].commands, 1)

	assert.Equal(t, "Other Commands", cmdGroups[3].category)
	assert.Len(t, cmdGroups[3].commands, 2)
}

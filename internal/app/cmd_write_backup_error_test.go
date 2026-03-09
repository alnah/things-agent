package app

import "testing"

func TestDestructiveCommandsReturnBackupErrorWhenDBIsMissing(t *testing.T) {
	fr := &fakeRunner{output: "ok"}
	setupTestRuntime(t, t.TempDir(), fr)

	cases := []struct {
		name string
		cmd  func() error
	}{
		{
			name: "delete-area",
			cmd: func() error {
				c := newDeleteAreaCmd()
				c.SetArgs([]string{"--name", "area"})
				return c.Execute()
			},
		},
		{
			name: "delete-project",
			cmd: func() error {
				c := newDeleteProjectCmd()
				c.SetArgs([]string{"--name", "proj"})
				return c.Execute()
			},
		},
		{
			name: "delete-task",
			cmd: func() error {
				c := newDeleteTaskCmd()
				c.SetArgs([]string{"--name", "task"})
				return c.Execute()
			},
		},
		{
			name: "tags delete",
			cmd: func() error {
				c := newTagsDeleteCmd()
				c.SetArgs([]string{"--name", "tag"})
				return c.Execute()
			},
		},
		{
			name: "delete-child-task",
			cmd: func() error {
				c := newDeleteChildTaskCmd()
				c.SetArgs([]string{"--parent", "task", "--name", "sub"})
				return c.Execute()
			},
		},
	}

	for _, tc := range cases {
		if err := tc.cmd(); err == nil {
			t.Fatalf("%s should fail without backupable db files", tc.name)
		}
	}
}

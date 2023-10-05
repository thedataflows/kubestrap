/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type Version struct {
	cmd    *cobra.Command
	parent *Root
}

var (
	_       = NewVersion(root)
	version = "dev"
)

func NewVersion(parent *Root) *Version {
	v := &Version{
		parent: parent,
	}

	v.cmd = &cobra.Command{
		Use:   "version",
		Short: "Display version and exit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}

	parent.Cmd().AddCommand(v.cmd)

	return v
}

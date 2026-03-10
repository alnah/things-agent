package command

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewMoveTaskCmd(runE func(*cobra.Command, []string, string, string, map[string]string) error) *cobra.Command {
	var name, id string
	var toArea, toAreaID, toProject, toProjectID, toHeading, toHeadingID string
	cmd := &cobra.Command{
		Use:   "move-task",
		Short: "Move a task to an area, project, or heading",
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := ResolveMoveTaskDestination(toArea, toAreaID, toProject, toProjectID, toHeading, toHeadingID)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, params)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&toArea, "to-area", "", "Target area name")
	cmd.Flags().StringVar(&toAreaID, "to-area-id", "", "Target area ID")
	cmd.Flags().StringVar(&toProject, "to-project", "", "Target project name")
	cmd.Flags().StringVar(&toProjectID, "to-project-id", "", "Target project ID")
	cmd.Flags().StringVar(&toHeading, "to-heading", "", "Target heading name")
	cmd.Flags().StringVar(&toHeadingID, "to-heading-id", "", "Target heading ID")
	return cmd
}

func NewMoveProjectCmd(runE func(*cobra.Command, []string, string, string, map[string]string) error) *cobra.Command {
	var name, id string
	var toArea, toAreaID string
	cmd := &cobra.Command{
		Use:   "move-project",
		Short: "Move a project to another area",
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := ResolveMoveProjectDestination(toArea, toAreaID)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, params)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&id, "id", "", "Project ID")
	cmd.Flags().StringVar(&toArea, "to-area", "", "Target area name")
	cmd.Flags().StringVar(&toAreaID, "to-area-id", "", "Target area ID")
	return cmd
}

func NewReorderProjectItemsCmd(runE func(*cobra.Command, []string, string, string, []string) error, parseCSV func(string) []string) *cobra.Command {
	var projectName, projectID, idsCSV string
	cmd := &cobra.Command{
		Use:   "reorder-project-items",
		Short: "Reorder tasks inside a project (private Things backend)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			projectName, projectID, err = ResolveEntitySelector(projectName, projectID)
			if err != nil {
				return err
			}
			ids := parseCSV(idsCSV)
			if len(ids) == 0 {
				return errors.New("--ids is required")
			}
			return runE(cmd, args, projectName, projectID, ids)
		},
	}
	cmd.Flags().StringVar(&projectName, "project", "", "Project name")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID")
	cmd.Flags().StringVar(&idsCSV, "ids", "", "Comma-separated ordered task IDs")
	return cmd
}

func NewReorderAreaItemsCmd(runE func(*cobra.Command, []string, string, string, []string) error, parseCSV func(string) []string) *cobra.Command {
	var areaName, areaID, idsCSV string
	cmd := &cobra.Command{
		Use:   "reorder-area-items",
		Short: "Reorder items inside an area (private Things backend)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			areaName, areaID, err = ResolveAreaSelector(areaName, areaID)
			if err != nil {
				return err
			}
			ids := parseCSV(idsCSV)
			if len(ids) == 0 {
				return errors.New("--ids is required")
			}
			return runE(cmd, args, areaName, areaID, ids)
		},
	}
	cmd.Flags().StringVar(&areaName, "area", "", "Area name")
	cmd.Flags().StringVar(&areaID, "area-id", "", "Area ID")
	cmd.Flags().StringVar(&idsCSV, "ids", "", "Comma-separated ordered item IDs")
	return cmd
}

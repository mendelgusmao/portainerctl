package cmd

import (
	"fmt"
	"strconv"

	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/config"
	"github.com/portainer/portainerctl/internal/output"
	"github.com/spf13/cobra"
)

// Structs match portaineree.Endpoint and portaineree.EndpointGroup from spec 2.39.1

type endpointAgentData struct {
	Version         string `json:"Version"`
	PreviousVersion string `json:"PreviousVersion"`
}

type endpoint struct {
	ID        int               `json:"Id"`
	Name      string            `json:"Name"`
	Type      int               `json:"Type"`
	URL       string            `json:"URL"`
	PublicURL string            `json:"PublicURL"`
	GroupID   int               `json:"GroupId"`
	Status    int               `json:"Status"`
	Agent     endpointAgentData `json:"Agent"`
	EdgeID    string            `json:"EdgeID"`
}

// portaineree.EndpointGroup (returned by /endpoint_groups)
type endpointGroup struct {
	ID          int    `json:"Id"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
}

func envTypeLabel(t int) string {
	switch t {
	case 1:
		return "docker-local"
	case 2:
		return "docker-agent"
	case 3:
		return "azure"
	case 4:
		return "k8s-agent"
	case 5:
		return "k8s-edge"
	case 6:
		return "docker-edge"
	case 7:
		return "podman-local"
	default:
		return strconv.Itoa(t)
	}
}

func envStatusLabel(s int) string {
	switch s {
	case 1:
		return "up"
	case 2:
		return "down"
	default:
		return "unknown"
	}
}

func envCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "env",
		Aliases: []string{"environment", "environments", "endpoint", "endpoints"},
		Short:   "Manage Portainer environments (endpoints)",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var envs []endpoint
			if err := c.Get("/endpoints", &envs); err != nil {
				return err
			}
			rows := [][]string{}
			for _, e := range envs {
				rows = append(rows, []string{
					strconv.Itoa(e.ID),
					e.Name,
					envTypeLabel(e.Type),
					envStatusLabel(e.Status),
					e.URL,
					strconv.Itoa(e.GroupID),
					e.Agent.Version,
				})
			}
			output.Table([]string{"ID", "NAME", "TYPE", "STATUS", "URL", "GROUP", "AGENT"}, rows)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get details of a specific environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var env map[string]interface{}
			if err := c.Get("/endpoints/"+args[0], &env); err != nil {
				return err
			}
			output.JSON(env)
			return nil
		},
	}

	useCmd := &cobra.Command{
		Use:   "use <id>",
		Short: "Use environment as default for subsequent commands",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			for index, ctx := range cfg.Contexts {
				if ctx.Name == cfg.CurrentContext {
					cfg.Contexts[index].DefaultEnvironmentID, _ = strconv.Atoi(args[0])
					break
				}
			}

			if err := config.Save(cfg); err != nil {
				return err
			}

			output.Success(fmt.Sprintf("Environment %s set as default for current context.", args[0]))
			return nil
		},
	}

	snapshotCmd := &cobra.Command{
		Use:   "snapshot <id>",
		Short: "Trigger a snapshot for a specific environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Post("/endpoints/"+args[0]+"/snapshot", nil, nil); err != nil {
				return err
			}
			output.Success("Snapshot triggered for environment " + args[0])
			return nil
		},
	}

	snapshotAllCmd := &cobra.Command{
		Use:   "snapshot-all",
		Short: "Trigger snapshots for all environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Post("/endpoints/snapshot", nil, nil); err != nil {
				return err
			}
			output.Success("Snapshot triggered for all environments.")
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/endpoints/" + args[0]); err != nil {
				return err
			}
			output.Success("Environment " + args[0] + " deleted.")
			return nil
		},
	}

	var bulkIDs []int
	deleteBulkCmd := &cobra.Command{
		Use:     "delete-bulk",
		Short:   "Delete multiple environments by ID",
		Example: "  portainerctl env delete-bulk --ids 3,7,12",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(bulkIDs) == 0 {
				return fmt.Errorf("--ids is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{"ids": bulkIDs}
			if err := c.Post("/endpoints/delete", body, nil); err != nil {
				return err
			}
			output.Success(fmt.Sprintf("Deleted %d environments.", len(bulkIDs)))
			return nil
		},
	}
	deleteBulkCmd.Flags().IntSliceVar(&bulkIDs, "ids", nil, "Comma-separated environment IDs to delete")

	relationsCmd := &cobra.Command{
		Use:   "relations",
		Short: "List environment relations (edge agent associations)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/endpoints/relations", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	agentVersionsCmd := &cobra.Command{
		Use:   "agent-versions",
		Short: "List available agent versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/endpoints/agent_versions", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	var settingsPatch string
	settingsCmd := &cobra.Command{
		Use:   "settings <id>",
		Short: "Get or update environment-level settings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if settingsPatch == "" {
				var result interface{}
				if err := c.Get("/endpoints/"+args[0]+"/settings", &result); err != nil {
					return err
				}
				output.JSON(result)
			} else {
				var body map[string]interface{}
				if err := parseJSON(settingsPatch, &body); err != nil {
					return err
				}
				var result interface{}
				if err := c.Put("/endpoints/"+args[0]+"/settings", body, &result); err != nil {
					return err
				}
				output.JSON(result)
			}
			return nil
		},
	}
	settingsCmd.Flags().StringVar(&settingsPatch, "patch", "", "JSON body to update settings (omit to read)")

	envRegistriesCmd := &cobra.Command{
		Use:   "registries <id>",
		Short: "List registries accessible to an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/endpoints/"+args[0]+"/registries", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, useCmd, snapshotCmd, snapshotAllCmd, deleteCmd, deleteBulkCmd,
		relationsCmd, agentVersionsCmd, settingsCmd, envRegistriesCmd)
	return cmd
}

func envGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env-group",
		Short: "Manage environment groups",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all environment groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			// Spec returns endpointgroups.endpointGroupResponse which has Id, Name, Description
			var groups []endpointGroup
			if err := c.Get("/endpoint_groups", &groups); err != nil {
				return err
			}
			rows := [][]string{}
			for _, g := range groups {
				rows = append(rows, []string{strconv.Itoa(g.ID), g.Name, g.Description})
			}
			output.Table([]string{"ID", "NAME", "DESCRIPTION"}, rows)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a specific environment group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/endpoint_groups/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	var groupName, groupDesc string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an environment group",
		RunE: func(cmd *cobra.Command, args []string) error {
			if groupName == "" {
				return fmt.Errorf("--name is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{"Name": groupName, "Description": groupDesc}
			var result interface{}
			if err := c.Post("/endpoint_groups", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&groupName, "name", "", "Group name")
	createCmd.Flags().StringVar(&groupDesc, "description", "", "Group description")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an environment group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/endpoint_groups/" + args[0]); err != nil {
				return err
			}
			output.Success("Environment group " + args[0] + " deleted.")
			return nil
		},
	}

	addEnvCmd := &cobra.Command{
		Use:   "add-env <group-id> <env-id>",
		Short: "Add an environment to a group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Post("/endpoint_groups/"+args[0]+"/endpoints/"+args[1], nil, nil); err != nil {
				return err
			}
			output.Success(fmt.Sprintf("Environment %s added to group %s.", args[1], args[0]))
			return nil
		},
	}

	removeEnvCmd := &cobra.Command{
		Use:   "remove-env <group-id> <env-id>",
		Short: "Remove an environment from a group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/endpoint_groups/" + args[0] + "/endpoints/" + args[1]); err != nil {
				return err
			}
			output.Success(fmt.Sprintf("Environment %s removed from group %s.", args[1], args[0]))
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, createCmd, deleteCmd, addEnvCmd, removeEnvCmd)
	return cmd
}

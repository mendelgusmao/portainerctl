package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/config"
	"github.com/portainer/portainerctl/internal/output"
	"github.com/spf13/cobra"
)

// portaineree.Stack — only scalar fields used in list table.
// GitConfig is gittypes.RepoConfig which has URL string at top level.
type gitRepoConfig struct {
	URL           string `json:"URL"`
	ReferenceName string `json:"ReferenceName"`
	ConfigFilePath string `json:"ConfigFilePath"`
}

type stack struct {
	ID         int            `json:"Id"`
	Name       string         `json:"Name"`
	Type       int            `json:"Type"`
	EndpointID int            `json:"EndpointId"`
	Status     int            `json:"Status"`
	CreatedBy  string         `json:"CreatedBy"`
	GitConfig  *gitRepoConfig `json:"GitConfig"`
}

func stackTypeLabel(t int) string {
	switch t {
	case 1:
		return "swarm"
	case 2:
		return "compose"
	case 3:
		return "kubernetes"
	default:
		return strconv.Itoa(t)
	}
}

func stackStatusLabel(s int) string {
	switch s {
	case 1:
		return "active"
	case 2:
		return "inactive"
	default:
		return "unknown"
	}
}

func mapStacks(c *client.Client, envId int) (map[string]string, error) {
	path := "/stacks"
	if envId > 0 {
		path += fmt.Sprintf("?filters={\"EndpointID\":%d}", envId)
	}
	var stacksList []stack
	if err := c.Get(path, &stacksList); err != nil {
		return nil, err
	}
	stacks := map[string]string{}

	for _, s := range stacksList {
		stacks[s.Name] = strconv.Itoa(s.ID)
	}
	return stacks, nil
}

func stackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Manage stacks (Compose, Swarm, Kubernetes manifests)",
	}

	var listEnvID int
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := "/stacks"
			if listEnvID > 0 {
				path += fmt.Sprintf("?filters={\"EndpointID\":%d}", listEnvID)
			}
			var stacks []stack
			if err := c.Get(path, &stacks); err != nil {
				return err
			}
			rows := [][]string{}
			for _, s := range stacks {
				git := ""
				if s.GitConfig != nil {
					git = s.GitConfig.URL
				}
				rows = append(rows, []string{
					strconv.Itoa(s.ID),
					s.Name,
					stackTypeLabel(s.Type),
					stackStatusLabel(s.Status),
					strconv.Itoa(s.EndpointID),
					s.CreatedBy,
					git,
				})
			}
			output.Table([]string{"ID", "NAME", "TYPE", "STATUS", "ENV", "CREATED BY", "GIT"}, rows)
			return nil
		},
	}
	listCmd.Flags().IntVar(&listEnvID, "env", 0, "Filter by environment ID")

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get stack details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/stacks/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	getByNameCmd := &cobra.Command{
		Use:   "get-by-name <n>",
		Short: "Get a stack by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/stacks/name/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	fileCmd := &cobra.Command{
		Use:   "file <id>",
		Short: "Get the compose/manifest file for a stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			// Response is stacks.stackFileResponse with StackFileContent string
			var result map[string]interface{}
			if err := c.Get("/stacks/"+args[0]+"/file", &result); err != nil {
				return err
			}
			if content, ok := result["StackFileContent"].(string); ok {
				fmt.Println(content)
			} else {
				output.JSON(result)
			}
			return nil
		},
	}

	var deployName, deployFile, deployRepo, deployBranch, deployPath string
	var deployEnvID int

	deployComposeCmd := &cobra.Command{
		Use:     "deploy-compose",
		Short:   "Deploy a Compose stack from a local file",
		Example: `  portainerctl stack deploy-compose --name myapp --env 2 --file docker-compose.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployFile == "" {
				return fmt.Errorf("--name, --env, and --file are required")
			}
			data, err := os.ReadFile(deployFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"Name":             deployName,
				"StackFileContent": string(data),
				"Env":              []interface{}{},
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/standalone/string?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployComposeCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployComposeCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target environment ID")
	deployComposeCmd.Flags().StringVar(&deployFile, "file", "", "Path to compose file")

	deploySwarmCmd := &cobra.Command{
		Use:     "deploy-swarm",
		Short:   "Deploy a Swarm stack from a local file",
		Example: `  portainerctl stack deploy-swarm --name myapp --env 2 --file docker-stack.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployFile == "" {
				return fmt.Errorf("--name, --env, and --file are required")
			}
			data, err := os.ReadFile(deployFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			// Fetch swarm ID from the environment's docker info
			var dockerInfo map[string]interface{}
			swarmID := ""
			if err := c.Get(fmt.Sprintf("/endpoints/%d/docker/info", deployEnvID), &dockerInfo); err == nil {
				if swarm, ok := dockerInfo["Swarm"].(map[string]interface{}); ok {
					if cluster, ok := swarm["Cluster"].(map[string]interface{}); ok {
						if id, ok := cluster["ID"].(string); ok {
							swarmID = id
						}
					}
				}
			}
			body := map[string]interface{}{
				"Name":             deployName,
				"StackFileContent": string(data),
				"SwarmID":          swarmID,
				"Env":              []interface{}{},
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/swarm/string?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deploySwarmCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deploySwarmCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target environment ID")
	deploySwarmCmd.Flags().StringVar(&deployFile, "file", "", "Path to stack file")

	deployGitCmd := &cobra.Command{
		Use:     "deploy-git",
		Short:   "Deploy a Compose stack from a Git repository",
		Example: `  portainerctl stack deploy-git --name myapp --env 2 --repo https://github.com/org/repo --branch main --path docker-compose.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployRepo == "" {
				return fmt.Errorf("--name, --env, and --repo are required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"Name":                     deployName,
				"RepositoryURL":            deployRepo,
				"RepositoryReferenceName":  "refs/heads/" + deployBranch,
				"ComposeFile":              deployPath,
				"Env":                      []interface{}{},
				"RepositoryAuthentication": false,
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/standalone/repository?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployGitCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployGitCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target environment ID")
	deployGitCmd.Flags().StringVar(&deployRepo, "repo", "", "Git repository URL")
	deployGitCmd.Flags().StringVar(&deployBranch, "branch", "main", "Git branch")
	deployGitCmd.Flags().StringVar(&deployPath, "path", "docker-compose.yml", "Compose file path in repo")

	deployK8sCmd := &cobra.Command{
		Use:     "deploy-k8s",
		Short:   "Deploy a Kubernetes manifest stack from a local file",
		Example: `  portainerctl stack deploy-k8s --name myapp --env 4 --file manifest.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployFile == "" {
				return fmt.Errorf("--name, --env, and --file are required")
			}
			data, err := os.ReadFile(deployFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"StackName":        deployName,
				"StackFileContent": string(data),
				"Namespace":        "default",
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/kubernetes/string?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployK8sCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployK8sCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target Kubernetes environment ID")
	deployK8sCmd.Flags().StringVar(&deployFile, "file", "", "Path to manifest file")

	deployK8sGitCmd := &cobra.Command{
		Use:     "deploy-k8s-git",
		Short:   "Deploy a Kubernetes stack from a Git repository",
		Example: `  portainerctl stack deploy-k8s-git --name myapp --env 4 --repo https://github.com/org/repo --branch main --path manifest.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployRepo == "" {
				return fmt.Errorf("--name, --env, and --repo are required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"StackName":                deployName,
				"RepositoryURL":            deployRepo,
				"RepositoryReferenceName":  "refs/heads/" + deployBranch,
				"ManifestFile":             deployPath,
				"RepositoryAuthentication": false,
				"Namespace":                "default",
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/kubernetes/repository?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployK8sGitCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployK8sGitCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target Kubernetes environment ID")
	deployK8sGitCmd.Flags().StringVar(&deployRepo, "repo", "", "Git repository URL")
	deployK8sGitCmd.Flags().StringVar(&deployBranch, "branch", "main", "Git branch")
	deployK8sGitCmd.Flags().StringVar(&deployPath, "path", "manifest.yaml", "Manifest file path in repo")

	var redeployEnvID int
	redeployCmd := &cobra.Command{
		Use:   "redeploy <id>",
		Short: "Redeploy a GitOps-backed stack (pull latest from Git)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := "/stacks/" + args[0] + "/git/redeploy"
			if redeployEnvID > 0 {
				path += fmt.Sprintf("?endpointId=%d", redeployEnvID)
			}
			var result interface{}
			if err := c.Put(path, map[string]interface{}{}, &result); err != nil {
				return err
			}
			output.Success("Stack " + args[0] + " redeployed from Git.")
			return nil
		},
	}
	redeployCmd.Flags().IntVar(&redeployEnvID, "env", 0, "Environment ID (required for stacks created before v1.18.0)")

	var startEnvID int
	startCmd := &cobra.Command{
		Use:   "start <id>",
		Short: "Start a stopped stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			ctx, err := cfg.Current()
			if err != nil {
				return err
			}

			if ctx.DefaultEnvironmentID == 0 && startEnvID == 0 {
				return fmt.Errorf("--env is required")
			} else if ctx.DefaultEnvironmentID > 0 && startEnvID == 0 {
				startEnvID = ctx.DefaultEnvironmentID
			}

			c, err := client.MustClient()
			if err != nil {
				return err
			}

			stackID := args[0]
			if _, err := strconv.Atoi(args[0]); err != nil {
				var ok bool
				stacks, err := mapStacks(c, startEnvID)
				if err != nil {
					return err
				}
				stackID, ok = stacks[args[0]]
				if !ok {
					return fmt.Errorf("stack not found: %s", args[0])
				}
			}

			var result interface{}
			path := fmt.Sprintf("/stacks/%s/start?endpointId=%d", stackID, startEnvID)
			if err := c.Post(path, nil, &result); err != nil {
				return err
			}
			output.Success("Stack " + args[0] + " started.")
			return nil
		},
	}
	startCmd.Flags().IntVar(&startEnvID, "env", 0, "Environment ID (required)")

	var restartEnvID int
	restartCmd := &cobra.Command{
		Use:   "restart <id>",
		Short: "Restart a stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			ctx, err := cfg.Current()
			if err != nil {
				return err
			}

			if ctx.DefaultEnvironmentID == 0 && restartEnvID == 0 {
				return fmt.Errorf("--env is required")
			} else if ctx.DefaultEnvironmentID > 0 && restartEnvID == 0 {
				restartEnvID = ctx.DefaultEnvironmentID
			}

			c, err := client.MustClient()
			if err != nil {
				return err
			}

			stackID := args[0]
			if _, err := strconv.Atoi(args[0]); err != nil {
				var ok bool
				stacks, err := mapStacks(c, startEnvID)
				if err != nil {
					return err
				}
				stackID, ok = stacks[args[0]]
				if !ok {
					return fmt.Errorf("stack not found: %s", args[0])
				}
			}

			var result interface{}
			path := fmt.Sprintf("/stacks/%s/start?endpointId=%d&forceRecreate=true", stackID, restartEnvID)
			if err := c.Post(path, nil, &result); err != nil {
				return err
			}
			output.Success("Stack " + args[0] + " restarted.")
			return nil
		},
	}
	restartCmd.Flags().IntVar(&restartEnvID, "env", 0, "Environment ID (required)")

	var stopEnvID int
	stopCmd := &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a running stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			ctx, err := cfg.Current()
			if err != nil {
				return err
			}

			if ctx.DefaultEnvironmentID == 0 && stopEnvID == 0 {
				return fmt.Errorf("--env is required")
			} else if ctx.DefaultEnvironmentID > 0 && stopEnvID == 0 {
				stopEnvID = ctx.DefaultEnvironmentID
			}

			c, err := client.MustClient()
			if err != nil {
				return err
			}

			stackID := args[0]
			if _, err := strconv.Atoi(args[0]); err != nil {
				var ok bool
				stacks, err := mapStacks(c, startEnvID)
				if err != nil {
					return err
				}
				stackID, ok = stacks[args[0]]
				if !ok {
					return fmt.Errorf("stack not found: %s", args[0])
				}
			}

			path := fmt.Sprintf("/stacks/%s/stop?endpointId=%d", stackID, stopEnvID)
			if err := c.Post(path, nil, nil); err != nil {
				return err
			}
			output.Success("Stack " + args[0] + " stopped.")
			return nil
		},
	}
	stopCmd.Flags().IntVar(&stopEnvID, "env", 0, "Environment ID (required)")

	var deleteEnvID int
	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deleteEnvID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/stacks/%s?endpointId=%d", args[0], deleteEnvID)
			if err := c.Delete(path); err != nil {
				return err
			}
			output.Success("Stack " + args[0] + " deleted.")
			return nil
		},
	}
	deleteCmd.Flags().IntVar(&deleteEnvID, "env", 0, "Environment ID (required)")

	imagesStatusCmd := &cobra.Command{
		Use:   "image-status <id>",
		Short: "Get image update status for a stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/stacks/"+args[0]+"/images_status", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, getByNameCmd, fileCmd,
		deployComposeCmd, deploySwarmCmd, deployGitCmd,
		deployK8sCmd, deployK8sGitCmd,
		redeployCmd, startCmd, restartCmd, stopCmd, deleteCmd, imagesStatusCmd)
	return cmd
}

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Manage pipeline templates",
}

var templatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all templates",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := daemon.SendCommand("templates", nil)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		templates, ok := data["templates"].([]interface{})
		if !ok || len(templates) == 0 {
			fmt.Println("No templates found")
			return
		}

		fmt.Printf("%-20s %-50s %s\n", "NAME", "DESCRIPTION", "STEPS")
		fmt.Println(strings.Repeat("-", 80))

		for _, t := range templates {
			tmpl := t.(map[string]interface{})
			steps := 0
			switch v := tmpl["steps"].(type) {
			case float64:
				steps = int(v)
			case int:
				steps = v
			}
			fmt.Printf("%-20s %-50s %d\n",
				tmpl["name"], tmpl["description"], steps)
		}
	},
}

var templatesShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show template details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := struct {
			Name string `json:"name"`
		}{Name: args[0]}

		resp, err := daemon.SendCommand("template-show", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		fmt.Printf("Template: %s\n", data["name"])
		fmt.Printf("Description: %s\n", data["description"])

		steps, ok := data["steps"].([]interface{})
		if ok {
			fmt.Println("\nSteps:")
			for _, s := range steps {
				step := s.(map[string]interface{})
				parallel := ""
				if p, ok := step["parallel"].(bool); ok && p {
					parallel = " (parallel)"
				}
				order := 0
				switch v := step["order"].(type) {
				case float64:
					order = int(v)
				case int:
					order = v
				}
				fmt.Printf("  %d. %s [%s]%s\n", order, step["name"], step["tool"], parallel)
			}
		}
	},
}

var templatesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new template from file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			fmt.Println("Error: --file flag is required")
			return
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			return
		}

		payload := struct {
			Name string `json:"name"`
			Data string `json:"data"`
		}{
			Name: args[0],
			Data: string(data),
		}

		resp, err := daemon.SendCommand("template-create", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		fmt.Printf("Template '%s' created\n", args[0])
	},
}

var templatesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a template",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := struct {
			Name string `json:"name"`
		}{Name: args[0]}

		resp, err := daemon.SendCommand("template-delete", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		fmt.Printf("Template '%s' deleted\n", args[0])
	},
}

func init() {
	templatesCreateCmd.Flags().StringP("file", "f", "", "Template YAML file path")

	templatesCmd.AddCommand(templatesListCmd)
	templatesCmd.AddCommand(templatesShowCmd)
	templatesCmd.AddCommand(templatesCreateCmd)
	templatesCmd.AddCommand(templatesDeleteCmd)
}

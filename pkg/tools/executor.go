package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Swarmind/libagent/internal/tools"
	"github.com/Swarmind/libagent/pkg/config"
	"github.com/rs/zerolog/log"

	"github.com/tmc/langchaingo/llms"
)

const CommandNotesPromptAddition = `Here is information about available commands usage preferences:`

var CommandExecutorDefinition = llms.FunctionDefinition{
	Name: "commandExecutor",
	Description: `Executes a provided string command in the bash -c wrapper.
You can use commands like 'tree' or 'ls' to view the file structure and discover file paths.
Uses temporary home directory.
Warning!
Every command will be executed from the temp home directory!
You need specifically chain cd before command if it needs to be launched in other than home directory!`,
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "the shell command to execute to",
			},
		},
	},
}

type CommandExecutorArgs struct {
	Command string `json:"command"`
}

// CommandExecutorTool represents a tool that executes commands using exec.Command.
type CommandExecutorTool struct {
	tempDir    *string
	sessionDir *string
}

// Call executes the command with the given arguments.
func (s *CommandExecutorTool) Call(ctx context.Context, input string) (string, error) {
	commandExecutorArgs := CommandExecutorArgs{}
	if err := json.Unmarshal([]byte(input), &commandExecutorArgs); err != nil {
		return "", err
	}

	if s.tempDir == nil {
		tDir, err := os.MkdirTemp("", "libagent_command_executor_session_")
		if err != nil {
			return "", err
		}
		log.Debug().Msgf("command executor temp directory %s created", tDir)
		s.tempDir = &tDir
	}

	log.Debug().Msgf("command executor: bash -c %s", commandExecutorArgs.Command)
	cmd := exec.Command("bash", "-c", commandExecutorArgs.Command)
	cmd.Dir = *s.tempDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%+v: %s", err, output)
	}

	return string(output), nil
}

func (s *CommandExecutorTool) cleanup() error {
	if s.tempDir == nil {
		return nil
	}

	log.Debug().Msgf("command executor remove temp directory %s", *s.tempDir)
	return os.RemoveAll(*s.tempDir)
}

func init() {
	globalToolsRegistry = append(globalToolsRegistry,
		func(ctx context.Context, cfg config.Config) (*tools.ToolData, error) {
			if cfg.CommandExecutorDisable {
				return nil, nil
			}

			commandExecutorTool := CommandExecutorTool{}

			definition := CommandExecutorDefinition
			if len(cfg.CommandExecutorCommands) > 0 {
				commandsList := ""
				for command, description := range cfg.CommandExecutorCommands {
					descSplit := strings.SplitN(description, "\n", 1)
					if len(descSplit) == 2 {
						commandsList += fmt.Sprintf(
							"- %s: %s\n%s\n",
							strings.ToLower(command),
							descSplit[0], descSplit[1],
						)
					} else {
						commandsList += fmt.Sprintf(
							"- %s: %s\n",
							strings.ToLower(command),
							description,
						)
					}
				}

				definition.Description += fmt.Sprintf(
					"\n%s\n%s",
					CommandNotesPromptAddition, commandsList,
				)
			}

			if strings.HasSuffix(definition.Description, "\n\n") {
				definition.Description = strings.TrimSuffix(definition.Description, "\n")
			}

			return &tools.ToolData{
				Definition: definition,
				Call:       commandExecutorTool.Call,
				Cleanup:    commandExecutorTool.cleanup,
			}, nil
		},
	)
}

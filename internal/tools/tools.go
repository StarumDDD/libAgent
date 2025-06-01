package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/tmc/langchaingo/llms"
)

var LLMDefinition = llms.FunctionDefinition{
	Name: "LLM",
	Description: `A pretrained LLM like yourself. Useful when you need to act with general
world knowledge and common sense. Prioritize it when you are confident in solving the problem
yourself. Input can be any instruction or task.`,
}

type ToolData struct {
	Definition llms.FunctionDefinition
	Call       func(context.Context, string) (string, error)
	Cleanup    func() error
}

type ToolsExecutor struct {
	Tools map[string]*ToolData
}

func (e ToolsExecutor) Execute(ctx context.Context, call llms.ToolCall) (llms.ToolCallResponse, error) {
	response := llms.ToolCallResponse{
		ToolCallID: call.ID,
		Name:       call.FunctionCall.Name,
	}

	content, err := e.CallTool(ctx,
		call.FunctionCall.Name,
		call.FunctionCall.Arguments,
	)
	if err != nil {
		return response, err
	}

	response.Content = content
	return response, err
}

func (e ToolsExecutor) GetTool(toolName string) (*ToolData, error) {
	toolData, ok := e.Tools[toolName]
	if !ok {
		return nil, fmt.Errorf("no such tool")
	}
	return toolData, nil
}

func (e ToolsExecutor) CallTool(ctx context.Context, toolName, args string) (string, error) {
	toolData, err := e.GetTool(toolName)
	if err != nil {
		return "", err
	}

	return toolData.Call(ctx, args)
}

func (e ToolsExecutor) ToolsList() []llms.Tool {
	tools := []llms.Tool{}
	for _, toolData := range e.Tools {
		tools = append(tools, llms.Tool{
			Type:     "function",
			Function: &toolData.Definition,
		})
	}
	slices.SortFunc(tools,
		func(a, b llms.Tool) int {
			return strings.Compare(a.Function.Name, b.Function.Name)
		},
	)

	return tools
}

func (e ToolsExecutor) ToolsPromptDesc() string {
	desc := ""

	funcDefs := []llms.FunctionDefinition{
		LLMDefinition,
	}
	for _, toolData := range e.Tools {
		funcDefs = append(funcDefs, toolData.Definition)
	}
	slices.SortFunc(funcDefs,
		func(a, b llms.FunctionDefinition) int {
			return strings.Compare(a.Name, b.Name)
		},
	)

	for idx, def := range funcDefs {
		input := "string"
		if def.Parameters != nil {
			if props, ok := def.Parameters.(map[string]interface{})["properties"]; ok {
				fields := map[string]string{}
				for prop, val := range props.(map[string]interface{}) {
					fields[prop] = val.(map[string]interface{})["type"].(string)
				}
				fieldsJson, err := json.Marshal(fields)
				if err == nil {
					input = string(fieldsJson)
				}
			}
		}
		desc += fmt.Sprintf("(%d) %s[%s]: %s\n", idx, def.Name, input, def.Description)
	}
	return desc
}

func (e ToolsExecutor) ProcessToolCalls(ctx context.Context, calls []llms.ToolCall) string {
	content := ""
	for _, toolCall := range calls {
		response, err := e.Execute(ctx, toolCall)
		if err != nil {
			log.Warn().Err(err).Msgf(
				"Tool %s call with args: %s",
				toolCall.FunctionCall.Name,
				toolCall.FunctionCall.Arguments,
			)
			content = fmt.Sprintf("Error calling tool %s with args: %s: %v",
				toolCall.FunctionCall.Name, toolCall.FunctionCall.Arguments, err,
			)
			continue
		}
		content = response.Content
	}
	return content
}

func (e ToolsExecutor) Cleanup() error {
	for _, tool := range e.Tools {
		if tool.Cleanup == nil {
			continue
		}
		if err := tool.Cleanup(); err != nil {
			return err
		}
	}
	return nil
}

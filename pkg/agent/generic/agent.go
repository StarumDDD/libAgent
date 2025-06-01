package generic

import (
	"context"

	"github.com/Swarmind/libagent/internal/tools"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Agent struct {
	LLM           *openai.LLM
	ToolsExecutor *tools.ToolsExecutor

	toolsList *[]llms.Tool
}

func (a *Agent) Run(
	ctx context.Context,
	state []llms.MessageContent,
	opts ...llms.CallOption,
) (llms.MessageContent, error) {
	if a.toolsList == nil {
		a.toolsList = &[]llms.Tool{}
	}
	if len(*a.toolsList) == 0 {
		*a.toolsList = a.ToolsExecutor.ToolsList()
	}

	opts = append(opts, llms.WithTools(*a.toolsList))

	response, err := a.LLM.GenerateContent(
		ctx, state, opts...,
	)
	if err != nil {
		return llms.MessageContent{}, err
	}

	content := response.Choices[0].Content
	if toolContent := a.ToolsExecutor.ProcessToolCalls(
		ctx, response.Choices[0].ToolCalls,
	); toolContent != "" {
		content = toolContent
	}

	return llms.TextParts(llms.ChatMessageTypeAI, content), nil
}

func (a *Agent) SimpleRun(
	ctx context.Context,
	input string,
	opts ...llms.CallOption,
) (string, error) {
	if a.toolsList == nil {
		a.toolsList = &[]llms.Tool{}
	}
	if len(*a.toolsList) == 0 {
		*a.toolsList = a.ToolsExecutor.ToolsList()
	}

	opts = append(opts, llms.WithTools(*a.toolsList))

	response, err := a.LLM.GenerateContent(ctx,
		[]llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman,
				input,
			)},
		opts...,
	)
	if err != nil {
		return "", err
	}

	content := response.Choices[0].Content
	if toolContent := a.ToolsExecutor.ProcessToolCalls(
		ctx, response.Choices[0].ToolCalls,
	); toolContent != "" {
		content = toolContent
	}

	return content, nil
}

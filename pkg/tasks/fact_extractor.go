package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	llms2 "github.com/tmc/langchaingo/llms"

	"github.com/getzep/zep/internal"
	"github.com/getzep/zep/pkg/llms"
	"github.com/getzep/zep/pkg/models"
)

const MaxTokensFallback = 2048
const FactMaxOutputTokens = 1024

var _ models.Task = &FactExtractorTask{}

// FactExtractorTask gets a list of messages created since the last SummaryPoint,
// determines if the message count exceeds the configured message window, and if
// so:
// - creates new and updates existing facts from the messages
// - The new SummaryPoint index will be created when the summary is generated. Add the facts to the summary.

type FactExtractorTask struct {
	BaseTask
}

func NewFactExtractorTask(appState *models.AppState) *FactExtractorTask {
	return &FactExtractorTask{
		BaseTask: BaseTask{
			appState: appState,
		},
	}
}

func (t *FactExtractorTask) Execute(
	ctx context.Context,
	msg *message.Message,
) error {
	ctx, done := context.WithTimeout(ctx, TaskTimeout*time.Second)
	defer done()

	sessionID := msg.Metadata.Get("session_id")
	if sessionID == "" {
		return errors.New("FactExtractorTask session_id is empty")
	}

	log.Debugf("FactExtractorTask called for session %s", sessionID)

	memory, err := t.appState.MemoryStore.GetMemory(
		ctx,
		sessionID,
		0,
	)

	if err != nil {
		return fmt.Errorf("FactExtractorTask GetMemory failed: %w", err)
	}

	messages := memory.Messages
	if messages == nil {
		log.Warningf("FactExtractorTask GetMemory returned no messages")
		return nil
	}
	newFacts, err := t.extractFacts(
		ctx,
		messages,
		memory.Summary,
		memory.Facts,
		0,
	)
	if err != nil {
		return fmt.Errorf("FactExtractorTask extractFacts failed: %w", err)
	}

	err = t.appState.MemoryStore.CreateFacts(
		ctx,
		sessionID,
		newFacts,
	)

	msg.Ack()

	return nil 
}

func (t *FactExtractorTask) extractFacts(
	ctx context.Context,
	messages []models.Message,
	summary *models.Summary,
	existingFacts []models.Fact,
	promptTokens int,
) ([]models.Fact, error) {
	var facts []models.Fact

	// TODO: LOOK INTO TOKENS

	// modelName, err := llms.GetLLMModelName(t.appState.Config)
	// if err != nil {
	// 	return []models.Fact{}, err
	// }
	// maxTokes, ok := llms.MaxLLMTokensMap[modelName]
	// if !ok {
	// 	maxTokes = MaxTokensFallback
	// }
	//
	// if promptTokens == 0 {
	// 	promptTokens = 250
	// }
	//

	// messagedJoined := strings.Join(messages, "\n")
	var messagesTexts []string
	for _, m := range messages {
		messagesText := fmt.Sprintf("%s: %s", m.Role, m.Content)
		messagesTexts = append(messagesTexts, messagesText)
	}

	var factsTexts []string
	for _, f := range existingFacts {
		factText := fmt.Sprintf("Fact: %s", f.Content)
		factsTexts = append(factsTexts, factText)
	}

	if len(messagesTexts) == 0 {
		return facts, nil
	}

	messagesJoined := strings.Join(messagesTexts, "\n")
	factsJoined := strings.Join(factsTexts, "\n")
	// prompt := defaultFactPromptTemplate

	promptData := FactPromptTemplateData{
		PrevFactsJoined: factsJoined,
		MessagesJoined:  messagesJoined,
	}

	parsedPrompt, err := internal.ParsePrompt(defaultFactPromptTemplate, promptData)
	if err != nil {
		return facts, fmt.Errorf("failed to parse prompt: %w", err)
	}

	newFacts, err := t.appState.LLMClient.Call(
		ctx,
		parsedPrompt,
		llms2.WithMaxTokens(FactMaxOutputTokens),
	)
	if err != nil {
		return facts, fmt.Errorf("failed to call LLM: %w", err)
	}

	newFacts = strings.TrimSpace(newFacts)
	parsedFacts := t.parseFinalOutput(newFacts)

	var finalFacts []models.Fact
	for _, fact := range parsedFacts {
		finalFacts = append(finalFacts, models.Fact{
			Content: fact,
		})
	}

	return finalFacts, nil
}

func (t *FactExtractorTask) parseFinalOutput(
	raw string,
) []string {
	lines := strings.Split(raw, "\n") // Split the input string by newline
	var facts []string                // Initialize an empty slice to hold the extracted facts

	for _, line := range lines {
		// Trim the "Fact: " prefix from each line
		if strings.HasPrefix(line, "Fact: ") {
			fact := strings.TrimPrefix(line, "Fact: ")
			facts = append(facts, fact) // Add the fact to the slice
		}
	}

	return facts // Return the slice of facts
}

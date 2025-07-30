package pdf_mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PromptManager handles conversion prompt templates
type PromptManager struct {
	prompts map[string]PromptTemplate
}

// PromptTemplate represents a conversion prompt template
type PromptTemplate struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	UseCase     string `json:"use_case"`
}

// ConversionSuggestion represents a suggested conversion approach
type ConversionSuggestion struct {
	RecommendedType  string   `json:"recommended_type"`
	Confidence       float64  `json:"confidence"`
	Reasoning        string   `json:"reasoning"`
	AlternativeTypes []string `json:"alternative_types,omitempty"`
}

func NewPromptManager() (*PromptManager, error) {
	pm := &PromptManager{
		prompts: make(map[string]PromptTemplate),
	}

	// Initialize default prompts
	pm.loadDefaultPrompts()

	return pm, nil
}

func (pm *PromptManager) loadDefaultPrompts() {
	// Handwritten notes prompt
	pm.prompts["handwritten"] = PromptTemplate{
		Type:        "handwritten",
		Name:        "Handwritten Notes",
		Description: "Optimized for converting handwritten notes and sketches",
		UseCase:     "Personal notes, meeting notes, sketches, diagrams",
		Prompt: `You are an expert at converting handwritten notes from images to well-structured Markdown format.

Please follow these guidelines:
1. Preserve all text content accurately, even if handwriting is messy
2. Use appropriate Markdown formatting (headers, lists, emphasis, etc.)
3. Structure the content logically with proper headings
4. If there are diagrams or drawings, describe them in [diagram: description] format
5. Maintain the original organization and flow of the notes
6. Use bullet points or numbered lists where appropriate
7. Bold important terms or concepts using **bold**
8. If handwriting is unclear, use [unclear] notation
9. Pay special attention to arrows, connections, and spatial relationships
10. Preserve any mathematical formulas or equations

Convert the handwritten content to clean, readable Markdown while preserving all meaningful information and the author's original structure.`,
	}

	// Typed document prompt
	pm.prompts["typed"] = PromptTemplate{
		Type:        "typed",
		Name:        "Typed Document",
		Description: "Optimized for converting typed documents and printed materials",
		UseCase:     "Articles, reports, printed documents, books",
		Prompt: `You are an expert at converting typed documents from images to well-structured Markdown format.

Please follow these guidelines:
1. Preserve all text content with high accuracy
2. Maintain original formatting structure (headers, paragraphs, lists)
3. Convert headers to appropriate Markdown levels (# ## ### etc.)
4. Preserve bullet points and numbered lists exactly
5. Maintain table structures if present using Markdown table syntax
6. Bold and italic text should be preserved using **bold** and *italic*
7. If there are images, charts, or figures, describe them in [figure: description] format
8. Preserve any code blocks or technical content with proper formatting
9. Maintain paragraph breaks and spacing
10. Keep any footnotes or references intact

Convert the typed content to clean, properly formatted Markdown while maintaining the document's professional structure.`,
	}

	// Mixed content prompt
	pm.prompts["mixed"] = PromptTemplate{
		Type:        "mixed",
		Name:        "Mixed Content",
		Description: "Optimized for documents with both text and visual elements",
		UseCase:     "Presentations, infographics, annotated documents",
		Prompt: `You are an expert at converting mixed-content documents from images to well-structured Markdown format.

Please follow these guidelines:
1. Identify and convert all text content accurately (both typed and handwritten)
2. Use appropriate Markdown formatting for different text types
3. Describe visual elements (charts, diagrams, images) in [visual: description] format
4. Maintain the logical flow between text and visual elements
5. Use headers to separate different sections or topics
6. For presentations, treat each major section as a separate heading
7. Convert bullet points and lists appropriately
8. If there are annotations or callouts, include them with context
9. Preserve any data tables using Markdown table syntax
10. Note relationships between text and visual elements

Convert the mixed content to comprehensive Markdown that captures both textual information and describes visual elements clearly.`,
	}

	// Research paper prompt
	pm.prompts["research"] = PromptTemplate{
		Type:        "research",
		Name:        "Research Paper",
		Description: "Optimized for academic papers and research documents",
		UseCase:     "Academic papers, research articles, technical documents",
		Prompt: `You are an expert at converting research documents from images to well-structured Markdown format.

Please follow these guidelines:
1. Preserve academic structure (Abstract, Introduction, Methods, Results, etc.)
2. Convert section headers to appropriate Markdown levels
3. Maintain citation formats and reference numbers
4. Preserve mathematical formulas and equations
5. Convert tables and figures with proper descriptions
6. Maintain footnotes and endnotes
7. Preserve author names, affiliations, and publication details
8. Keep technical terminology and jargon intact
9. Maintain paragraph structure and academic writing flow
10. Note any graphs, charts, or data visualizations with detailed descriptions

Convert the research content to properly formatted academic Markdown suitable for scholarly use.`,
	}
}

func (pm *PromptManager) GetPrompt(promptType string) (PromptTemplate, error) {
	prompt, exists := pm.prompts[promptType]
	if !exists {
		return PromptTemplate{}, fmt.Errorf("prompt type '%s' not found", promptType)
	}
	return prompt, nil
}

func (pm *PromptManager) GetAllPrompts() map[string]PromptTemplate {
	return pm.prompts
}

func (pm *PromptManager) ListPromptTypes() []string {
	types := make([]string, 0, len(pm.prompts))
	for t := range pm.prompts {
		types = append(types, t)
	}
	return types
}

func (pm *PromptManager) GetPromptsAsJSON() (string, error) {
	data, err := json.MarshalIndent(pm.prompts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal prompts: %w", err)
	}
	return string(data), nil
}

// SuggestPromptType analyzes file metadata and suggests the best prompt type
func (pm *PromptManager) SuggestPromptType(fileName string, fileSize int64) ConversionSuggestion {
	fileName = strings.ToLower(fileName)

	// Simple heuristics for suggestion
	if strings.Contains(fileName, "note") || strings.Contains(fileName, "sketch") {
		return ConversionSuggestion{
			RecommendedType:  "handwritten",
			Confidence:       0.7,
			Reasoning:        "Filename suggests handwritten notes or sketches",
			AlternativeTypes: []string{"mixed"},
		}
	}

	if strings.Contains(fileName, "paper") || strings.Contains(fileName, "research") || strings.Contains(fileName, "journal") {
		return ConversionSuggestion{
			RecommendedType:  "research",
			Confidence:       0.8,
			Reasoning:        "Filename suggests academic or research content",
			AlternativeTypes: []string{"typed"},
		}
	}

	if strings.Contains(fileName, "presentation") || strings.Contains(fileName, "slide") {
		return ConversionSuggestion{
			RecommendedType:  "mixed",
			Confidence:       0.8,
			Reasoning:        "Filename suggests presentation with mixed content",
			AlternativeTypes: []string{"typed"},
		}
	}

	// Default suggestion based on file size
	if fileSize > 5*1024*1024 { // > 5MB
		return ConversionSuggestion{
			RecommendedType:  "mixed",
			Confidence:       0.6,
			Reasoning:        "Large file size suggests complex document with mixed content",
			AlternativeTypes: []string{"typed", "research"},
		}
	}

	// Default to typed for most documents
	return ConversionSuggestion{
		RecommendedType:  "typed",
		Confidence:       0.5,
		Reasoning:        "Default suggestion for typical document",
		AlternativeTypes: []string{"mixed", "handwritten"},
	}
}

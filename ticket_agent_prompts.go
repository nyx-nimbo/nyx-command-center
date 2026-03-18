package main

import (
	"fmt"
	"strings"
)

// Prompt templates for the ticket agent's AI interactions

const ticketAnalysisPrompt = `You are a senior software engineer analyzing a development ticket.
Break down the ticket into concrete implementation steps.

For each step, identify:
1. What needs to be changed or created
2. Which files are likely involved (based on the project stack and conventions)
3. Any potential risks or dependencies
4. Estimated effort for the step

Provide your analysis as structured text that another developer (or AI coding agent) can follow to implement the ticket from start to finish.

Be specific about file paths, function names, and data structures where possible.
Do not include code — just the implementation plan.`

const ticketWorkPrompt = `You are an autonomous coding agent working on a development ticket.
You will receive the ticket details and project context.
Your job is to analyze what needs to be done and produce a detailed implementation plan
with specific instructions for making the code changes.

Include:
- Files to create or modify
- Specific changes needed in each file
- Any new dependencies or imports required
- Database schema changes if applicable
- API endpoint changes if applicable
- Frontend changes if applicable

Respond with a clear, structured plan that includes the reasoning behind each change.
Format your response as markdown with sections for each component of the change.`

const ticketPRDescriptionPrompt = `You are a senior engineer writing a pull request description.
Given the ticket details and work notes, generate a clear, professional PR description.

The PR description should include:
## Summary
A concise summary of what was changed and why.

## Changes
A bulleted list of the specific changes made.

## How to Test
Step-by-step instructions for reviewing and testing the changes.

## Notes
Any additional context, trade-offs, or follow-up items.

Keep it concise but thorough. Use technical language appropriate for a code review.`

// buildTicketWorkPrompt constructs the full prompt for the AI to work on a ticket,
// interpolating the ticket and project data into the template.
func buildTicketWorkPrompt(ticket Ticket, project Project) string {
	var b strings.Builder

	b.WriteString(ticketWorkPrompt)
	b.WriteString("\n\n---\n\n")

	b.WriteString(fmt.Sprintf("## Project: %s\n", project.Name))
	if project.Description != "" {
		b.WriteString(fmt.Sprintf("**Description:** %s\n", project.Description))
	}
	if project.Stack != "" {
		b.WriteString(fmt.Sprintf("**Tech Stack:** %s\n", project.Stack))
	}
	if project.RepoURL != "" {
		b.WriteString(fmt.Sprintf("**Repository:** %s\n", project.RepoURL))
	}

	b.WriteString(fmt.Sprintf("\n## Ticket: %s — %s\n", ticket.Code, ticket.Title))
	b.WriteString(fmt.Sprintf("**Type:** %s | **Priority:** %s | **Estimate:** %s\n",
		ticket.Type, ticket.Priority, ticket.Estimate))

	if ticket.Description != "" {
		b.WriteString(fmt.Sprintf("\n### Description\n%s\n", ticket.Description))
	}
	if ticket.Scope != "" {
		b.WriteString(fmt.Sprintf("\n### Scope\n%s\n", ticket.Scope))
	}
	if len(ticket.AcceptanceCriteria) > 0 {
		b.WriteString("\n### Acceptance Criteria\n")
		for _, ac := range ticket.AcceptanceCriteria {
			b.WriteString(fmt.Sprintf("- %s\n", ac))
		}
	}
	if ticket.TechnicalNotes != "" {
		b.WriteString(fmt.Sprintf("\n### Technical Notes\n%s\n", ticket.TechnicalNotes))
	}

	b.WriteString("\n---\n\nPlease analyze this ticket and provide a detailed implementation plan.\n")

	return b.String()
}

// buildTicketAnalysisPrompt constructs the prompt for analyzing a ticket's requirements
// and breaking them down into implementation steps.
func buildTicketAnalysisPrompt(ticket Ticket) string {
	var b strings.Builder

	b.WriteString(ticketAnalysisPrompt)
	b.WriteString("\n\n---\n\n")

	b.WriteString(fmt.Sprintf("## Ticket: %s — %s\n", ticket.Code, ticket.Title))
	b.WriteString(fmt.Sprintf("**Type:** %s | **Priority:** %s\n", ticket.Type, ticket.Priority))

	if ticket.Description != "" {
		b.WriteString(fmt.Sprintf("\n### Description\n%s\n", ticket.Description))
	}
	if ticket.Scope != "" {
		b.WriteString(fmt.Sprintf("\n### Scope\n%s\n", ticket.Scope))
	}
	if len(ticket.AcceptanceCriteria) > 0 {
		b.WriteString("\n### Acceptance Criteria\n")
		for _, ac := range ticket.AcceptanceCriteria {
			b.WriteString(fmt.Sprintf("- %s\n", ac))
		}
	}
	if ticket.TechnicalNotes != "" {
		b.WriteString(fmt.Sprintf("\n### Technical Notes\n%s\n", ticket.TechnicalNotes))
	}

	b.WriteString("\n---\n\nProvide a step-by-step implementation breakdown for this ticket.\n")

	return b.String()
}

// buildPRDescriptionPrompt constructs the prompt for generating a PR description
// from the ticket details and work notes produced by the coding agent.
func buildPRDescriptionPrompt(ticket Ticket, workNotes string) string {
	var b strings.Builder

	b.WriteString(ticketPRDescriptionPrompt)
	b.WriteString("\n\n---\n\n")

	b.WriteString(fmt.Sprintf("## Ticket: %s — %s\n", ticket.Code, ticket.Title))

	if ticket.Description != "" {
		b.WriteString(fmt.Sprintf("\n### Ticket Description\n%s\n", ticket.Description))
	}
	if len(ticket.AcceptanceCriteria) > 0 {
		b.WriteString("\n### Acceptance Criteria\n")
		for _, ac := range ticket.AcceptanceCriteria {
			b.WriteString(fmt.Sprintf("- %s\n", ac))
		}
	}

	if workNotes != "" {
		b.WriteString(fmt.Sprintf("\n### Work Notes / Agent Output\n%s\n", workNotes))
	}

	b.WriteString("\n---\n\nGenerate the PR description now.\n")

	return b.String()
}

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// --- Gmail Types ---

// EmailMessage represents an email in the inbox list
type EmailMessage struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	Subject string `json:"subject"`
	Snippet string `json:"snippet"`
	Date    string `json:"date"`
	IsRead  bool   `json:"isRead"`
}

// EmailDetail represents a full email with body
type EmailDetail struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Date    string `json:"date"`
	IsRead  bool   `json:"isRead"`
}

// EmailResult wraps email list responses
type EmailResult struct {
	Emails []EmailMessage `json:"emails"`
	Error  string         `json:"error,omitempty"`
}

// EmailDetailResult wraps a single email response
type EmailDetailResult struct {
	Email EmailDetail `json:"email"`
	Error string      `json:"error,omitempty"`
}

// SendEmailResult wraps send email response
type SendEmailResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// --- Calendar Types ---

// CalendarEvent represents a calendar event
type CalendarEvent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Color       string `json:"color"`
	AllDay      bool   `json:"allDay"`
}

// CalendarResult wraps calendar event responses
type CalendarResult struct {
	Events []CalendarEvent `json:"events"`
	Error  string          `json:"error,omitempty"`
}

// CreateEventResult wraps create event response
type CreateEventResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// DeleteEventResult wraps delete event response
type DeleteEventResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// --- Gmail Service ---

func (a *App) getGmailService() (*gmail.Service, error) {
	token, err := loadToken()
	if err != nil {
		return nil, fmt.Errorf("no Google token found: %v", err)
	}

	// Refresh token if expired
	if token.Expiry.Before(time.Now()) {
		tokenSource := googleOAuthConfig.TokenSource(context.Background(), token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %v", err)
		}
		token = newToken
		_ = saveToken(token)
	}

	client := googleOAuthConfig.Client(context.Background(), token)
	srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %v", err)
	}
	return srv, nil
}

// GetEmails returns inbox messages
func (a *App) GetEmails(limit int) EmailResult {
	srv, err := a.getGmailService()
	if err != nil {
		return EmailResult{Error: err.Error()}
	}

	if limit <= 0 {
		limit = 20
	}

	loc, _ := time.LoadLocation("America/Mexico_City")

	msgs, err := srv.Users.Messages.List("me").
		LabelIds("INBOX").
		MaxResults(int64(limit)).
		Do()
	if err != nil {
		return EmailResult{Error: fmt.Sprintf("failed to list messages: %v", err)}
	}

	var emails []EmailMessage
	for _, m := range msgs.Messages {
		msg, err := srv.Users.Messages.Get("me", m.Id).
			Format("metadata").
			MetadataHeaders("From", "Subject", "Date").
			Do()
		if err != nil {
			continue
		}

		email := EmailMessage{
			ID:      msg.Id,
			Snippet: msg.Snippet,
			IsRead:  !containsLabel(msg.LabelIds, "UNREAD"),
		}

		for _, h := range msg.Payload.Headers {
			switch h.Name {
			case "From":
				email.From = parseFromHeader(h.Value)
			case "Subject":
				email.Subject = h.Value
			case "Date":
				email.Date = formatEmailDate(h.Value, loc)
			}
		}

		emails = append(emails, email)
	}

	if emails == nil {
		emails = []EmailMessage{}
	}

	return EmailResult{Emails: emails}
}

// GetEmail returns a full email with body
func (a *App) GetEmail(id string) EmailDetailResult {
	srv, err := a.getGmailService()
	if err != nil {
		return EmailDetailResult{Error: err.Error()}
	}

	loc, _ := time.LoadLocation("America/Mexico_City")

	msg, err := srv.Users.Messages.Get("me", id).Format("full").Do()
	if err != nil {
		return EmailDetailResult{Error: fmt.Sprintf("failed to get message: %v", err)}
	}

	detail := EmailDetail{
		ID:     msg.Id,
		IsRead: !containsLabel(msg.LabelIds, "UNREAD"),
	}

	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "From":
			detail.From = h.Value
		case "To":
			detail.To = h.Value
		case "Subject":
			detail.Subject = h.Value
		case "Date":
			detail.Date = formatEmailDate(h.Value, loc)
		}
	}

	detail.Body = extractBody(msg.Payload)

	return EmailDetailResult{Email: detail}
}

// SendEmail sends an email
func (a *App) SendEmail(to, subject, body string) SendEmailResult {
	srv, err := a.getGmailService()
	if err != nil {
		return SendEmailResult{Error: err.Error()}
	}

	// Get sender email
	profile, err := srv.Users.GetProfile("me").Do()
	if err != nil {
		return SendEmailResult{Error: fmt.Sprintf("failed to get profile: %v", err)}
	}

	raw := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s",
		profile.EmailAddress, to, subject, body)

	encoded := base64.URLEncoding.EncodeToString([]byte(raw))

	gmailMsg := &gmail.Message{
		Raw: encoded,
	}

	_, err = srv.Users.Messages.Send("me", gmailMsg).Do()
	if err != nil {
		return SendEmailResult{Error: fmt.Sprintf("failed to send: %v", err)}
	}

	return SendEmailResult{Success: true}
}

// MarkAsRead marks an email as read
func (a *App) MarkAsRead(id string) SendEmailResult {
	srv, err := a.getGmailService()
	if err != nil {
		return SendEmailResult{Error: err.Error()}
	}

	_, err = srv.Users.Messages.Modify("me", id, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}).Do()
	if err != nil {
		return SendEmailResult{Error: fmt.Sprintf("failed to mark as read: %v", err)}
	}

	return SendEmailResult{Success: true}
}

// --- Calendar Service ---

func (a *App) getCalendarService() (*calendar.Service, error) {
	token, err := loadToken()
	if err != nil {
		return nil, fmt.Errorf("no Google token found: %v", err)
	}

	// Refresh token if expired
	if token.Expiry.Before(time.Now()) {
		tokenSource := googleOAuthConfig.TokenSource(context.Background(), token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %v", err)
		}
		token = newToken
		_ = saveToken(token)
	}

	client := googleOAuthConfig.Client(context.Background(), token)
	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Calendar service: %v", err)
	}
	return srv, nil
}

// GetTodayEvents returns today's calendar events
func (a *App) GetTodayEvents() CalendarResult {
	loc, _ := time.LoadLocation("America/Mexico_City")
	now := time.Now().In(loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	return a.getEventsInRange(startOfDay, endOfDay)
}

// GetUpcomingEvents returns events for the next N days (excluding today)
func (a *App) GetUpcomingEvents(days int) CalendarResult {
	if days <= 0 {
		days = 7
	}

	loc, _ := time.LoadLocation("America/Mexico_City")
	now := time.Now().In(loc)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	endDate := time.Date(now.Year(), now.Month(), now.Day()+days+1, 0, 0, 0, 0, loc)

	return a.getEventsInRange(tomorrow, endDate)
}

func (a *App) getEventsInRange(start, end time.Time) CalendarResult {
	srv, err := a.getCalendarService()
	if err != nil {
		return CalendarResult{Error: err.Error()}
	}

	events, err := srv.Events.List("primary").
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		return CalendarResult{Error: fmt.Sprintf("failed to list events: %v", err)}
	}

	loc, _ := time.LoadLocation("America/Mexico_City")
	var result []CalendarEvent
	colorMap := []string{"accent", "success", "warning", "info", "accent", "success", "warning", "info"}

	for i, item := range events.Items {
		ev := CalendarEvent{
			ID:          item.Id,
			Title:       item.Summary,
			Location:    item.Location,
			Description: item.Description,
			Color:       colorMap[i%len(colorMap)],
		}

		if item.Start.DateTime != "" {
			t, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			ev.StartTime = t.In(loc).Format("2006-01-02T15:04:05")
			ev.AllDay = false
		} else if item.Start.Date != "" {
			ev.StartTime = item.Start.Date + "T00:00:00"
			ev.AllDay = true
		}

		if item.End.DateTime != "" {
			t, _ := time.Parse(time.RFC3339, item.End.DateTime)
			ev.EndTime = t.In(loc).Format("2006-01-02T15:04:05")
		} else if item.End.Date != "" {
			ev.EndTime = item.End.Date + "T23:59:59"
		}

		// Use calendar color if available
		if item.ColorId != "" {
			ev.Color = calendarColorToCSS(item.ColorId)
		}

		result = append(result, ev)
	}

	if result == nil {
		result = []CalendarEvent{}
	}

	return CalendarResult{Events: result}
}

// CreateEvent creates a new calendar event
func (a *App) CreateEvent(title, description, startTime, endTime string) CreateEventResult {
	srv, err := a.getCalendarService()
	if err != nil {
		return CreateEventResult{Error: err.Error()}
	}

	loc, _ := time.LoadLocation("America/Mexico_City")

	start, err := time.ParseInLocation("2006-01-02T15:04", startTime, loc)
	if err != nil {
		return CreateEventResult{Error: fmt.Sprintf("invalid start time: %v", err)}
	}

	end, err := time.ParseInLocation("2006-01-02T15:04", endTime, loc)
	if err != nil {
		return CreateEventResult{Error: fmt.Sprintf("invalid end time: %v", err)}
	}

	event := &calendar.Event{
		Summary:     title,
		Description: description,
		Start: &calendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
			TimeZone: "America/Mexico_City",
		},
		End: &calendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
			TimeZone: "America/Mexico_City",
		},
	}

	created, err := srv.Events.Insert("primary", event).Do()
	if err != nil {
		return CreateEventResult{Error: fmt.Sprintf("failed to create event: %v", err)}
	}

	return CreateEventResult{Success: true, ID: created.Id}
}

// DeleteEvent deletes a calendar event
func (a *App) DeleteEvent(id string) DeleteEventResult {
	srv, err := a.getCalendarService()
	if err != nil {
		return DeleteEventResult{Error: err.Error()}
	}

	err = srv.Events.Delete("primary", id).Do()
	if err != nil {
		return DeleteEventResult{Error: fmt.Sprintf("failed to delete event: %v", err)}
	}

	return DeleteEventResult{Success: true}
}

// --- Helper functions ---

func containsLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}

func parseFromHeader(from string) string {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		// Fallback: return as-is but try to clean up
		if idx := strings.Index(from, "<"); idx > 0 {
			return strings.TrimSpace(from[:idx])
		}
		return from
	}
	if addr.Name != "" {
		return addr.Name
	}
	return addr.Address
}

func formatEmailDate(dateStr string, loc *time.Location) string {
	// Try common email date formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
		"2 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	}

	var t time.Time
	var err error
	for _, f := range formats {
		t, err = time.Parse(f, dateStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return dateStr
	}

	t = t.In(loc)
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)

	if t.After(today) {
		return t.Format("3:04 PM")
	} else if t.After(yesterday) {
		return "Yesterday"
	} else if t.Year() == now.Year() {
		return t.Format("Jan 2")
	}
	return t.Format("Jan 2, 2006")
}

func extractBody(payload *gmail.MessagePart) string {
	// Try to find text/plain first, then text/html
	if payload.MimeType == "text/plain" && payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	// Check parts recursively
	var plainText, htmlText string
	for _, part := range payload.Parts {
		body := extractBodyFromPart(part, &plainText, &htmlText)
		if body != "" {
			return body
		}
	}

	if plainText != "" {
		return plainText
	}
	if htmlText != "" {
		return htmlText
	}

	// Fallback to top-level body
	if payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	return ""
}

func extractBodyFromPart(part *gmail.MessagePart, plainText, htmlText *string) string {
	if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			*plainText = string(data)
			return string(data)
		}
	}

	if part.MimeType == "text/html" && part.Body != nil && part.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			*htmlText = string(data)
		}
	}

	// Recurse into multipart
	for _, sub := range part.Parts {
		body := extractBodyFromPart(sub, plainText, htmlText)
		if body != "" {
			return body
		}
	}

	return ""
}

func calendarColorToCSS(colorID string) string {
	switch colorID {
	case "1":
		return "info" // lavender
	case "2":
		return "success" // sage
	case "3":
		return "accent" // grape
	case "4":
		return "warning" // flamingo
	case "5":
		return "warning" // banana
	case "6":
		return "warning" // tangerine
	case "7":
		return "info" // peacock
	case "8":
		return "accent" // graphite
	case "9":
		return "info" // blueberry
	case "10":
		return "success" // basil
	case "11":
		return "warning" // tomato
	default:
		return "accent"
	}
}

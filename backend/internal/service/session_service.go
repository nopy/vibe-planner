package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrInvalidSessionStatus = errors.New("invalid session status")
	ErrOpenCodeAPICall      = errors.New("opencode API call failed")
	ErrSessionAlreadyActive = errors.New("session already active for this task")
)

type SessionService interface {
	StartSession(ctx context.Context, taskID uuid.UUID, prompt string) (*model.Session, error)
	StopSession(ctx context.Context, sessionID uuid.UUID) error
	GetSession(ctx context.Context, sessionID uuid.UUID) (*model.Session, error)
	GetSessionsByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Session, error)
	GetActiveProjectSessions(ctx context.Context, projectID uuid.UUID) ([]model.Session, error)
	GetAllActiveSessions(ctx context.Context) ([]model.Session, error)
	UpdateSessionOutput(ctx context.Context, sessionID uuid.UUID, output string) error
	UpdateSessionStatus(ctx context.Context, sessionID uuid.UUID, status string, errorMsg string) error
	UpdateLastEventID(ctx context.Context, sessionID uuid.UUID, lastEventID string) error
}

type sessionService struct {
	sessionRepo   repository.SessionRepository
	taskRepo      repository.TaskRepository
	projectRepo   repository.ProjectRepository
	k8sService    KubernetesService
	configService ConfigServiceInterface
	sharedSecret  string
	httpClient    *http.Client
}

func NewSessionService(
	sessionRepo repository.SessionRepository,
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	k8sService KubernetesService,
	configService ConfigServiceInterface,
	sharedSecret string,
) SessionService {
	return &sessionService{
		sessionRepo:   sessionRepo,
		taskRepo:      taskRepo,
		projectRepo:   projectRepo,
		k8sService:    k8sService,
		configService: configService,
		sharedSecret:  sharedSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *sessionService) StartSession(ctx context.Context, taskID uuid.UUID, prompt string) (*model.Session, error) {
	// Get task and verify it exists
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Get project to resolve pod IP
	project, err := s.projectRepo.FindByID(ctx, task.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Check if task already has an active session
	activeSessions, err := s.sessionRepo.FindActiveSessionsForProject(ctx, project.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active sessions: %w", err)
	}

	for _, session := range activeSessions {
		if session.TaskID == taskID {
			return nil, ErrSessionAlreadyActive
		}
	}

	// Get pod IP from Kubernetes
	podIP, err := s.k8sService.GetPodIP(ctx, project.PodName, project.PodNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod IP: %w", err)
	}

	// Create session record in database
	session := &model.Session{
		TaskID:    taskID,
		ProjectID: project.ID,
		Status:    model.SessionStatusPending,
		Prompt:    prompt,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Start OpenCode session on sidecar
	startedAt := time.Now()
	remoteSessionID, err := s.callOpenCodeStart(ctx, podIP, session.ID, prompt, project.ID)
	if err != nil {
		session.Status = model.SessionStatusFailed
		session.Error = err.Error()
		_ = s.sessionRepo.Update(ctx, session)
		return nil, fmt.Errorf("%w: %v", ErrOpenCodeAPICall, err)
	}

	session.Status = model.SessionStatusRunning
	session.StartedAt = &startedAt
	session.RemoteSessionID = remoteSessionID
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session status: %w", err)
	}

	return session, nil
}

func (s *sessionService) StopSession(ctx context.Context, sessionID uuid.UUID) error {
	// Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Only stop if session is active
	if session.Status != model.SessionStatusPending && session.Status != model.SessionStatusRunning {
		return fmt.Errorf("%w: cannot stop session with status %s", ErrInvalidSessionStatus, session.Status)
	}

	// Get project to resolve pod IP
	project, err := s.projectRepo.FindByID(ctx, session.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	podIP, err := s.k8sService.GetPodIP(ctx, project.PodName, project.PodNamespace)
	if err != nil {
		return fmt.Errorf("failed to get pod IP: %w", err)
	}

	// Call OpenCode stop endpoint
	if err := s.callOpenCodeStop(ctx, podIP, sessionID); err != nil {
		// Log error but still update database status
		return fmt.Errorf("%w: %v", ErrOpenCodeAPICall, err)
	}

	// Update session status to cancelled
	session.Status = model.SessionStatusCancelled
	completedAt := time.Now()
	session.CompletedAt = &completedAt

	if session.StartedAt != nil {
		session.DurationMs = completedAt.Sub(*session.StartedAt).Milliseconds()
	}

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (s *sessionService) GetSession(ctx context.Context, sessionID uuid.UUID) (*model.Session, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

func (s *sessionService) GetSessionsByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Session, error) {
	sessions, err := s.sessionRepo.FindByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by task ID: %w", err)
	}

	return sessions, nil
}

func (s *sessionService) GetActiveProjectSessions(ctx context.Context, projectID uuid.UUID) ([]model.Session, error) {
	sessions, err := s.sessionRepo.FindActiveSessionsForProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active project sessions: %w", err)
	}

	return sessions, nil
}

func (s *sessionService) UpdateSessionOutput(ctx context.Context, sessionID uuid.UUID, output string) error {
	if err := s.sessionRepo.UpdateOutput(ctx, sessionID, output); err != nil {
		return fmt.Errorf("failed to update session output: %w", err)
	}

	return nil
}

// callOpenCodeStart starts a new OpenCode session on the sidecar
func (s *sessionService) callOpenCodeStart(ctx context.Context, podIP string, sessionID uuid.UUID, prompt string, projectID uuid.UUID) (string, error) {
	url := fmt.Sprintf("http://%s:3003/sessions", podIP)

	config, err := s.configService.GetActiveConfig(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("failed to get project config: %w", err)
	}

	apiKey, err := s.configService.GetDecryptedAPIKey(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	requestBody := map[string]interface{}{
		"session_id": sessionID.String(),
		"prompt":     prompt,
		"model_config": map[string]interface{}{
			"provider":      config.ModelProvider,
			"model":         config.ModelName,
			"api_key":       apiKey,
			"temperature":   config.Temperature,
			"max_tokens":    config.MaxTokens,
			"enabled_tools": config.EnabledTools,
		},
	}

	if config.ModelVersion != nil {
		requestBody["model_config"].(map[string]interface{})["model_version"] = *config.ModelVersion
	}
	if config.APIEndpoint != nil {
		requestBody["model_config"].(map[string]interface{})["api_endpoint"] = *config.APIEndpoint
	}
	if config.SystemPrompt != nil {
		requestBody["system_prompt"] = *config.SystemPrompt
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if s.sharedSecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.sharedSecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenCode API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenCode API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		SessionID       string `json:"session_id"`
		RemoteSessionID string `json:"remote_session_id"`
		Status          string `json:"status"`
		CreatedAt       string `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.RemoteSessionID, nil
}

// callOpenCodeStop stops an active OpenCode session on the sidecar
func (s *sessionService) callOpenCodeStop(ctx context.Context, podIP string, sessionID uuid.UUID) error {
	url := fmt.Sprintf("http://%s:3003/sessions/%s/stop", podIP, sessionID.String())

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if s.sharedSecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.sharedSecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call OpenCode API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenCode API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *sessionService) GetAllActiveSessions(ctx context.Context) ([]model.Session, error) {
	return s.sessionRepo.FindAllActiveSessions(ctx)
}

func (s *sessionService) UpdateSessionStatus(ctx context.Context, sessionID uuid.UUID, status string, errorMsg string) error {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.Status = model.SessionStatus(status)
	if errorMsg != "" {
		session.Output += fmt.Sprintf("\nError: %s\n", errorMsg)
	}

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (s *sessionService) UpdateLastEventID(ctx context.Context, sessionID uuid.UUID, lastEventID string) error {
	if err := s.sessionRepo.UpdateLastEventID(ctx, sessionID, lastEventID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("failed to update last event ID: %w", err)
	}

	return nil
}

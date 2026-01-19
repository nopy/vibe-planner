# OpenCode Project Manager - TODO List

**Last Updated:** 2026-01-19 18:45 CET  
**Current Phase:** Phase 7 - Two-Way Interactions (Weeks 13-14)  
**Status:** 🚧 READY TO START - Phase 7 Planning  
**Branch:** main

---

## ✅ Phases 1-6: COMPLETE

🎉 **All foundational phases complete** - Ready for Phase 7 (Two-Way Interactions)!

See archived phases:
- [PHASE1.md](./PHASE1.md) - OIDC Authentication (Complete 2026-01-16)
- [PHASE2.md](./PHASE2.md) - Project Management with Kubernetes (Complete 2026-01-18)
- [PHASE3.md](./PHASE3.md) - Task Management & Kanban Board (Complete 2026-01-19 00:45)
- [PHASE4.md](./PHASE4.md) - File Explorer with Monaco Editor (Complete 2026-01-19 12:25)
- [PHASE5.md](./PHASE5.md) - OpenCode Integration & Execution (Complete 2026-01-19 14:56)
- [PHASE6.md](./PHASE6.md) - OpenCode Configuration UI (Complete 2026-01-19 18:31)

**Total Project Stats:**
- ✅ **383 tests passing** (231 backend + 152 frontend)
- ✅ **6 phases complete** (Auth, Projects, Tasks, Files, Execution, Config)
- ✅ **Production-ready features:** Authentication, CRUD, real-time updates, file editing, config management
- ✅ **Next:** Enable chat-like interaction with AI during task execution

---

## 🚧 Phase 7: Two-Way Interactions (Weeks 13-14)

**Objective:** Enable users to provide feedback, ask questions, and guide the AI agent during task execution through bidirectional communication.

**Status:** 📋 Planning (Ready to Start)

**Key Features:**
- User can send messages to AI during task execution
- AI can ask clarifying questions and wait for user response
- Interaction history persisted and displayable
- WebSocket connection for real-time bidirectional communication
- Integration with existing task execution flow

---

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  Frontend (React)                                               │
│  ├─ InteractionPanel (chat interface in Task Detail)           │
│  ├─ MessageList (conversation history)                         │
│  ├─ MessageInput (user input with send button)                 │
│  └─ TypingIndicator (agent is thinking...)                     │
└─────────────────┬───────────────────────────────────────────────┘
                  │ WebSocket (bidirectional)
┌─────────────────▼───────────────────────────────────────────────┐
│  Backend API (Go)                                               │
│  ├─ WebSocket Handler (task/:id/interact)                      │
│  ├─ Interaction Service (message routing)                      │
│  ├─ Session Proxy (forward to OpenCode server)                 │
│  └─ Message Queue (buffer during execution)                    │
└─────────────────┬───────────────────────────────────────────────┘
                  │ read/write
┌─────────────────▼───────────────────────────────────────────────┐
│  PostgreSQL Database                                            │
│  ├─ interactions (message history)                             │
│  └─ Foreign key: task_id → tasks.id                            │
└─────────────────────────────────────────────────────────────────┘
```

**Key Design Decisions:**
1. **WebSocket Communication:** Use WebSocket for real-time bidirectional messaging (not SSE)
2. **Message Persistence:** Store all user-AI interactions in database for history/audit
3. **Integration Point:** Hook into existing task execution flow (Phase 5)
4. **Session Proxy:** Route messages through session-proxy sidecar to OpenCode server
5. **Message Types:** Support user messages, AI responses, system notifications, status updates

---

### Backend Tasks

#### 7.1 Interaction Model & Repository

**Status:** 📋 Planned

**Objective:** Create database schema and repository layer for user-AI interactions.

**Tasks:**
1. **Create Interaction Migration (`db/migrations/006_add_interactions.up.sql`):**
   ```sql
   CREATE TABLE interactions (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
       session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
       user_id UUID NOT NULL REFERENCES users(id),
       
       -- Message content
       message_type VARCHAR(50) NOT NULL,  -- user_message, agent_response, system_notification
       content TEXT NOT NULL,
       metadata JSONB,  -- Additional context (e.g., code snippets, file references)
       
       -- Timestamps
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       
       -- Indexing
       CHECK (message_type IN ('user_message', 'agent_response', 'system_notification'))
   );
   
   CREATE INDEX idx_interactions_task_id ON interactions(task_id);
   CREATE INDEX idx_interactions_session_id ON interactions(session_id);
   CREATE INDEX idx_interactions_created_at ON interactions(created_at);
   ```

2. **Create Interaction Model (`backend/internal/model/interaction.go`):**
   ```go
   package model
   
   import (
       "time"
       "github.com/google/uuid"
   )
   
   type Interaction struct {
       ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
       TaskID      uuid.UUID `gorm:"type:uuid;not null;index" json:"task_id"`
       SessionID   *uuid.UUID `gorm:"type:uuid;index" json:"session_id,omitempty"`
       UserID      uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
       
       MessageType string    `gorm:"size:50;not null" json:"message_type"`  // user_message, agent_response, system_notification
       Content     string    `gorm:"type:text;not null" json:"content"`
       Metadata    JSONB     `gorm:"type:jsonb" json:"metadata,omitempty"`
       
       CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
   }
   
   func (Interaction) TableName() string {
       return "interactions"
   }
   ```

3. **Create Interaction Repository (`backend/internal/repository/interaction_repository.go`):**
   ```go
   package repository
   
   import (
       "context"
       "github.com/google/uuid"
       "github.com/npinot/vibe/backend/internal/model"
       "gorm.io/gorm"
   )
   
   type InteractionRepository struct {
       db *gorm.DB
   }
   
   func NewInteractionRepository(db *gorm.DB) *InteractionRepository {
       return &InteractionRepository{db: db}
   }
   
   // CreateInteraction stores a new interaction message
   func (r *InteractionRepository) CreateInteraction(ctx context.Context, interaction *model.Interaction) error
   
   // GetInteractionsByTask retrieves all interactions for a task (ordered by created_at)
   func (r *InteractionRepository) GetInteractionsByTask(ctx context.Context, taskID uuid.UUID) ([]model.Interaction, error)
   
   // GetInteractionsBySession retrieves all interactions for a session
   func (r *InteractionRepository) GetInteractionsBySession(ctx context.Context, sessionID uuid.UUID) ([]model.Interaction, error)
   
   // DeleteInteractionsByTask deletes all interactions for a task (cascade handled by DB)
   func (r *InteractionRepository) DeleteInteractionsByTask(ctx context.Context, taskID uuid.UUID) error
   ```

4. **Create Repository Tests (`backend/internal/repository/interaction_repository_test.go`):**
   - Test `CreateInteraction()` stores message correctly
   - Test `GetInteractionsByTask()` returns ordered list
   - Test `GetInteractionsBySession()` filters by session
   - Test foreign key constraints (task deletion → cascade)
   - Test message type validation (only allowed types)
   - **Target:** 15-20 unit tests

**Files to Create:**
- `db/migrations/006_add_interactions.up.sql`
- `db/migrations/006_add_interactions.down.sql`
- `backend/internal/model/interaction.go`
- `backend/internal/repository/interaction_repository.go`
- `backend/internal/repository/interaction_repository_test.go`

**Dependencies:**
- Existing Task and Session models
- JSONB type from config model (reuse)

**Success Criteria:**
- [ ] Migration runs successfully
- [ ] Repository tests pass (target: 15-20)
- [ ] Foreign key constraints enforced
- [ ] Message ordering correct (oldest first)

---

#### 7.2 Interaction Service

**Status:** 📋 Planned

**Objective:** Business logic for managing interactions and routing messages.

**Tasks:**
1. **Create Interaction Service (`backend/internal/service/interaction_service.go`):**
   ```go
   package service
   
   import (
       "context"
       "fmt"
       
       "github.com/google/uuid"
       "github.com/npinot/vibe/backend/internal/model"
       "github.com/npinot/vibe/backend/internal/repository"
   )
   
   type InteractionService struct {
       interactionRepo *repository.InteractionRepository
       sessionService  *SessionService  // For session validation
   }
   
   func NewInteractionService(
       interactionRepo *repository.InteractionRepository,
       sessionService *SessionService,
   ) *InteractionService
   
   // CreateUserMessage stores a user message and forwards to OpenCode
   func (s *InteractionService) CreateUserMessage(
       ctx context.Context,
       taskID uuid.UUID,
       userID uuid.UUID,
       content string,
   ) (*model.Interaction, error)
   
   // CreateAgentResponse stores an agent response message
   func (s *InteractionService) CreateAgentResponse(
       ctx context.Context,
       taskID uuid.UUID,
       sessionID uuid.UUID,
       content string,
   ) (*model.Interaction, error)
   
   // GetTaskHistory retrieves all interactions for a task
   func (s *InteractionService) GetTaskHistory(
       ctx context.Context,
       taskID uuid.UUID,
   ) ([]model.Interaction, error)
   
   // ValidateTaskOwnership ensures user can interact with task
   func (s *InteractionService) ValidateTaskOwnership(
       ctx context.Context,
       taskID uuid.UUID,
       userID uuid.UUID,
   ) error
   ```

2. **Create Service Tests (`backend/internal/service/interaction_service_test.go`):**
   - Test `CreateUserMessage()` stores and forwards message
   - Test `CreateAgentResponse()` stores response
   - Test `GetTaskHistory()` returns ordered history
   - Test `ValidateTaskOwnership()` enforces security
   - Test error handling (invalid task, unauthorized user)
   - **Target:** 15-20 unit tests

**Files to Create:**
- `backend/internal/service/interaction_service.go`
- `backend/internal/service/interaction_service_test.go`

**Success Criteria:**
- [ ] Service tests pass (target: 15-20)
- [ ] Task ownership validation working
- [ ] Message routing logic correct

---

#### 7.3 WebSocket Interaction Endpoint

**Status:** 📋 Planned

**Objective:** Real-time bidirectional communication endpoint for user-AI interaction.

**Tasks:**
1. **Create Interaction Handler (`backend/internal/api/interactions.go`):**
   ```go
   package api
   
   import (
       "net/http"
       
       "github.com/gin-gonic/gin"
       "github.com/google/uuid"
       "github.com/gorilla/websocket"
       "github.com/npinot/vibe/backend/internal/service"
   )
   
   type InteractionHandler struct {
       interactionService *service.InteractionService
   }
   
   func NewInteractionHandler(interactionService *service.InteractionService) *InteractionHandler
   
   // TaskInteractionWebSocket handles bidirectional communication for a task
   // WebSocket endpoint: /api/tasks/:id/interact
   func (h *InteractionHandler) TaskInteractionWebSocket(c *gin.Context)
   
   // GetTaskHistory retrieves all interactions for a task
   // HTTP endpoint: GET /api/tasks/:id/interactions
   func (h *InteractionHandler) GetTaskHistory(c *gin.Context)
   ```

2. **WebSocket Message Protocol:**
   ```json
   // Client → Server (user message)
   {
     "type": "user_message",
     "content": "Can you add error handling to this function?"
   }
   
   // Server → Client (agent response)
   {
     "type": "agent_response",
     "content": "I've added try-catch blocks to handle potential errors..."
   }
   
   // Server → Client (status update)
   {
     "type": "status_update",
     "content": "Agent is analyzing code..."
   }
   
   // Server → Client (system notification)
   {
     "type": "system_notification",
     "content": "Task execution completed"
   }
   ```

3. **Create API Handler Tests (`backend/internal/api/interactions_test.go`):**
   - Test WebSocket upgrade
   - Test user message handling
   - Test agent response broadcasting
   - Test authentication required
   - Test task ownership validation
   - Test concurrent connections
   - **Target:** 20-25 unit tests

**Files to Create:**
- `backend/internal/api/interactions.go`
- `backend/internal/api/interactions_test.go`

**Files to Modify:**
- `backend/cmd/api/main.go` (register routes)

**Success Criteria:**
- [ ] WebSocket handler tests pass (target: 20-25)
- [ ] Bidirectional communication working
- [ ] Authentication enforced
- [ ] Concurrent connections supported

---

#### 7.4 Integration with Task Execution

**Status:** 📋 Planned

**Objective:** Hook interaction system into existing task execution flow (Phase 5).

**Tasks:**
1. **Modify Task Execution Service:**
   - Add interaction callback to `ExecuteTaskWithOpenCode()`
   - Stream agent questions/responses through interaction service
   - Pause execution when agent asks for user input
   - Resume execution when user responds

2. **Update Session Proxy Sidecar:**
   - Add `/interact` endpoint for bidirectional messaging
   - Forward user messages to OpenCode server
   - Broadcast agent responses back to main API

3. **Create Integration Tests:**
   - Test end-to-end interaction flow (user message → agent → response)
   - Test execution pause/resume on user input
   - Test interaction history persistence

**Files to Modify:**
- `backend/internal/service/task_service.go` (add interaction callback)
- `sidecars/session-proxy/internal/handler/interact.go` (new file)

**Files to Create:**
- `backend/internal/api/interactions_integration_test.go`

**Success Criteria:**
- [ ] Integration tests pass
- [ ] Task execution pauses on agent questions
- [ ] User responses resume execution
- [ ] Interaction history persisted correctly

---

### Frontend Tasks

#### 7.5 Interaction Types & API Client

**Status:** 📋 Planned

**Objective:** TypeScript types and API client methods for interactions.

**Tasks:**
1. **Add Interaction Types (`frontend/src/types/index.ts`):**
   ```typescript
   export interface Interaction {
     id: string;
     task_id: string;
     session_id?: string;
     user_id: string;
     message_type: 'user_message' | 'agent_response' | 'system_notification';
     content: string;
     metadata?: Record<string, unknown>;
     created_at: string;
   }
   
   export interface InteractionMessage {
     type: 'user_message' | 'agent_response' | 'status_update' | 'system_notification';
     content: string;
     metadata?: Record<string, unknown>;
   }
   ```

2. **Add Interaction API Methods (`frontend/src/services/api.ts`):**
   ```typescript
   // Get interaction history for a task
   export const getTaskInteractions = async (taskId: string): Promise<Interaction[]>
   
   // Note: WebSocket handled separately via useInteractions hook
   ```

**Files to Modify:**
- `frontend/src/types/index.ts`
- `frontend/src/services/api.ts`

**Success Criteria:**
- [ ] TypeScript types defined
- [ ] API methods implemented
- [ ] Type safety verified

---

#### 7.6 useInteractions Hook

**Status:** 📋 Planned

**Objective:** Custom React hook for WebSocket-based interaction management.

**Tasks:**
1. **Create useInteractions Hook (`frontend/src/hooks/useInteractions.ts`):**
   ```typescript
   import { useState, useEffect, useRef, useCallback } from 'react';
   import type { Interaction, InteractionMessage } from '@/types';
   
   interface UseInteractionsReturn {
     messages: Interaction[];
     isConnected: boolean;
     isTyping: boolean;
     error: string | null;
     sendMessage: (content: string) => void;
     reconnect: () => void;
   }
   
   export const useInteractions = (taskId: string): UseInteractionsReturn => {
     const [messages, setMessages] = useState<Interaction[]>([]);
     const [isConnected, setIsConnected] = useState(false);
     const [isTyping, setIsTyping] = useState(false);
     const [error, setError] = useState<string | null>(null);
     const wsRef = useRef<WebSocket | null>(null);
     
     // WebSocket connection logic
     // Message handling
     // Auto-reconnection with exponential backoff
     // Typing indicator management
   };
   ```

2. **Key Features:**
   - WebSocket connection management
   - Message history loading on connect
   - Auto-reconnection with exponential backoff (same as useTaskExecution)
   - Typing indicator (agent is thinking...)
   - Error handling and recovery

**Files to Create:**
- `frontend/src/hooks/useInteractions.ts`
- `frontend/src/hooks/__tests__/useInteractions.test.ts`

**Success Criteria:**
- [ ] Hook connects to WebSocket successfully
- [ ] Messages sent and received correctly
- [ ] Typing indicator working
- [ ] Auto-reconnection functional
- [ ] Hook tests pass (target: 10-15)

---

#### 7.7 InteractionPanel Component

**Status:** 📋 Planned

**Objective:** Chat interface component for task detail page.

**Tasks:**
1. **Create InteractionPanel (`frontend/src/components/Kanban/InteractionPanel.tsx`):**
   ```typescript
   import React from 'react';
   import { MessageList } from './MessageList';
   import { MessageInput } from './MessageInput';
   import { TypingIndicator } from './TypingIndicator';
   import { useInteractions } from '@/hooks/useInteractions';
   
   interface InteractionPanelProps {
     taskId: string;
   }
   
   export const InteractionPanel: React.FC<InteractionPanelProps> = ({ taskId }) => {
     const { messages, isConnected, isTyping, sendMessage } = useInteractions(taskId);
     
     return (
       <div className="flex flex-col h-full">
         <MessageList messages={messages} />
         {isTyping && <TypingIndicator />}
         <MessageInput onSend={sendMessage} disabled={!isConnected} />
       </div>
     );
   };
   ```

2. **Key Features:**
   - Chat-like interface (message bubbles)
   - User messages: Right-aligned, blue background
   - Agent messages: Left-aligned, gray background
   - System notifications: Center-aligned, italic text
   - Auto-scroll to latest message
   - Connection status indicator

**Files to Create:**
- `frontend/src/components/Kanban/InteractionPanel.tsx`

**Success Criteria:**
- [ ] Component renders without errors
- [ ] Messages display correctly (user vs agent styling)
- [ ] Auto-scroll to latest message working
- [ ] Connection status visible

---

#### 7.8 MessageList, MessageInput, TypingIndicator Components

**Status:** 📋 Planned

**Objective:** Sub-components for interaction panel.

**Tasks:**
1. **MessageList Component:**
   - Displays message history with timestamps
   - User vs Agent message styling
   - Auto-scroll to bottom on new message
   - "Load more" pagination (if history > 100 messages)

2. **MessageInput Component:**
   - Text input with send button
   - Enter to send, Shift+Enter for new line
   - Character limit indicator (e.g., 0/2000)
   - Disabled state when not connected

3. **TypingIndicator Component:**
   - Animated dots ("Agent is thinking...")
   - Appears when agent is processing
   - Similar to chat apps (Slack, Discord)

**Files to Create:**
- `frontend/src/components/Kanban/MessageList.tsx`
- `frontend/src/components/Kanban/MessageInput.tsx`
- `frontend/src/components/Kanban/TypingIndicator.tsx`

**Success Criteria:**
- [ ] All three components render correctly
- [ ] MessageInput validates character limit
- [ ] TypingIndicator animates smoothly
- [ ] MessageList auto-scrolls correctly

---

#### 7.9 Integration with Task Detail Page

**Status:** 📋 Planned

**Objective:** Add interaction panel to existing task detail UI.

**Tasks:**
1. **Modify TaskDetailPanel:**
   - Add "Interact" tab next to "Details" and "Output"
   - Show unread message count badge (if any)
   - Preserve existing tabs (Details, Output)

2. **Add to TaskDetailPanel Layout:**
   ```typescript
   // frontend/src/components/Kanban/TaskDetailPanel.tsx
   
   <div className="flex border-b">
     <button onClick={() => setTab('details')}>Details</button>
     <button onClick={() => setTab('output')}>Output</button>
     <button onClick={() => setTab('interact')}>
       Interact {unreadCount > 0 && <span className="badge">{unreadCount}</span>}
     </button>
   </div>
   
   {tab === 'interact' && <InteractionPanel taskId={task.id} />}
   ```

**Files to Modify:**
- `frontend/src/components/Kanban/TaskDetailPanel.tsx`

**Success Criteria:**
- [ ] Interact tab visible in task detail
- [ ] Tab switching works correctly
- [ ] Unread badge shows correctly
- [ ] InteractionPanel renders in tab

---

### Testing Tasks

#### 7.10 Backend Unit Tests

**Status:** 📋 Planned

**Objective:** Comprehensive test coverage for all interaction layers.

**Target Tests:**
- Interaction Repository: 15-20 tests
- Interaction Service: 15-20 tests
- Interaction API Handlers: 20-25 tests
- **Total:** 50-65 backend unit tests

**Success Criteria:**
- [ ] All backend tests pass (target: 50-65)
- [ ] No regressions in existing tests (231 backend tests still passing)

---

#### 7.11 Frontend Component Tests

**Status:** 📋 Planned

**Objective:** Test all interaction UI components.

**Target Tests:**
- useInteractions hook: 10-15 tests
- InteractionPanel: 10 tests
- MessageList: 8 tests
- MessageInput: 8 tests
- TypingIndicator: 5 tests
- **Total:** 41-46 frontend tests

**Success Criteria:**
- [ ] All frontend tests pass (target: 41-46)
- [ ] >80% code coverage for interaction components
- [ ] No regressions in existing tests (152 frontend tests still passing)

---

#### 7.12 Integration Tests

**Status:** 📋 Planned

**Objective:** End-to-end tests for interaction workflow.

**Test Scenarios:**
1. **Complete Interaction Flow:**
   - User sends message via WebSocket
   - Message persisted in database
   - Agent receives message (via session proxy)
   - Agent responds
   - Response persisted and broadcast to user
   - History retrievable via HTTP API

2. **Concurrent Interactions:**
   - Multiple users interacting with different tasks
   - No message leakage between tasks
   - Correct user attribution

**Files to Create:**
- `backend/internal/api/interactions_integration_test.go`

**Success Criteria:**
- [ ] Integration tests pass
- [ ] No message leakage between tasks
- [ ] Interaction history persisted correctly

---

### Documentation

#### 7.13 API Documentation

**Status:** 📋 Planned

**Objective:** Document all interaction endpoints in API_SPECIFICATION.md.

**Endpoints to Document:**
- WebSocket `/api/tasks/:id/interact` - Bidirectional messaging
- GET `/api/tasks/:id/interactions` - Get interaction history

**Success Criteria:**
- [ ] All endpoints documented with examples
- [ ] WebSocket protocol documented
- [ ] Message types explained

---

## Success Criteria (Phase 7)

**Backend:** 🎯 Target
- [ ] Migration 006 (interactions) applied successfully
- [ ] Interaction repository tests: 15-20 passing
- [ ] Interaction service tests: 15-20 passing
- [ ] Interaction API handler tests: 20-25 passing
- [ ] Integration tests: 2+ passing (interaction flow + concurrent)

**Frontend:** 🎯 Target
- [ ] useInteractions hook functional and tested
- [ ] InteractionPanel component working
- [ ] MessageList, MessageInput, TypingIndicator components working
- [ ] Integration with TaskDetailPanel complete
- [ ] Component tests: 41-46 passing (>80% coverage)

**Integration:** 🎯 Target
- [ ] End-to-end interaction flow tested
- [ ] WebSocket bidirectional communication working
- [ ] Interaction history persisted correctly
- [ ] Task execution integration tested

**Documentation:** 🎯 Target
- [ ] API endpoints documented
- [ ] WebSocket protocol documented
- [ ] Integration guide updated

---

## Deferred Items (Phase 7 → Future)

1. **Rich Message Formatting**  
   **Impact:** Better UX for code snippets, file diffs, links  
   **Effort:** Medium (4-6 hours)  
   **Priority:** Low  
   **Deferred to:** Phase 10 (Polish)

2. **File Attachment in Messages**  
   **Impact:** Users can share files/screenshots with agent  
   **Effort:** High (2-3 days)  
   **Priority:** Low  
   **Deferred to:** Future enhancement

3. **Voice Input/Output**  
   **Impact:** Accessibility, mobile-friendly interaction  
   **Effort:** Very High (1-2 weeks)  
   **Priority:** Low  
   **Deferred to:** Future enhancement

4. **Multi-Agent Conversations**  
   **Impact:** Multiple AI agents collaborate on single task  
   **Effort:** Very High (2-3 weeks)  
   **Priority:** Low  
   **Deferred to:** Future enhancement

---

## Notes

**WebSocket Reconnection:**
- Use exponential backoff (same as useTaskExecution hook)
- Max retry: 5 attempts
- Backoff: 1s, 2s, 4s, 8s, 16s

**Message Character Limit:**
- User messages: 2,000 characters (prevent abuse)
- Agent responses: No limit (AI can be verbose)

**Typing Indicator:**
- Show when agent sends "status_update" message
- Auto-hide after 5 seconds if no response

**Interaction History Pagination:**
- Default: Last 100 messages
- "Load more" button loads next 100 (oldest first)

---

**Phase 7 Start Date:** TBD  
**Target Completion:** TBD (flexible, 3-developer team)  
**Author:** Sisyphus (OpenCode AI Agent)

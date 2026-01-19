export interface User {
  id: string
  oidc_subject: string
  email: string
  name?: string
  picture_url?: string
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface Project {
  id: string
  user_id: string
  name: string
  slug: string
  description?: string
  pod_name?: string
  pod_namespace?: string
  pod_status?: string
  workspace_pvc_name?: string
  status: 'initializing' | 'ready' | 'error' | 'archived'
  created_at: string
  updated_at: string
}

export type TaskStatus = 'todo' | 'in_progress' | 'ai_review' | 'human_review' | 'done'
export type TaskPriority = 'low' | 'medium' | 'high'

export interface Task {
  id: string
  project_id: string
  title: string
  description?: string
  status: TaskStatus
  position: number
  priority: TaskPriority
  assigned_to?: string
  current_session_id?: string
  opencode_output?: string
  execution_duration_ms: number
  file_references?: Record<string, unknown>
  created_by: string
  created_at: string
  updated_at: string
  deleted_at?: string
}

export interface OpenCodeConfig {
  id: string
  project_id: string
  version: number
  is_active: boolean
  
  // Model configuration
  model_provider: string
  model_name: string
  model_version?: string
  
  // Provider configuration
  api_endpoint?: string
  api_key?: string // Only for create/update requests
  temperature: number
  max_tokens: number
  
  // Tools configuration
  enabled_tools: string[]
  tools_config?: Record<string, unknown>
  
  // System configuration
  system_prompt?: string
  max_iterations: number
  timeout_seconds: number
  
  // Metadata
  created_by: string
  created_at: string
  updated_at: string
}

export interface CreateConfigRequest {
  model_provider: string
  model_name: string
  model_version?: string
  api_endpoint?: string
  api_key?: string
  temperature: number
  max_tokens: number
  enabled_tools: string[]
  tools_config?: Record<string, unknown>
  system_prompt?: string
  max_iterations: number
  timeout_seconds: number
}

export interface OpenCodeSession {
  id: string
  project_id: string
  task_id?: string
  remote_session_id?: string
  status: string
  prompt: string
  final_output?: string
  exit_code?: number
  execution_start_at?: string
  execution_end_at?: string
  created_at: string
  updated_at: string
}

export type PodStatus = 'Pending' | 'Running' | 'Succeeded' | 'Failed' | 'Unknown'

export type MessageType = 'user_message' | 'agent_response' | 'system_notification'
export type WebSocketMessageType = MessageType | 'status_update' | 'error' | 'history'

export interface Interaction {
  id: string
  task_id: string
  session_id?: string
  user_id: string
  message_type: MessageType
  content: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface InteractionMessage {
  type: WebSocketMessageType
  content: string
  metadata?: Record<string, unknown>
  timestamp: string
}

export interface InteractionHistoryResponse {
  interactions: Interaction[]
}

export interface CreateProjectRequest {
  name: string
  description?: string
  repo_url?: string
}

export interface UpdateProjectRequest {
  name?: string
  description?: string
  repo_url?: string
}

export interface CreateTaskRequest {
  title: string
  description?: string
  priority?: TaskPriority
}

export interface UpdateTaskRequest {
  title?: string
  description?: string
  priority?: TaskPriority
}

export interface MoveTaskRequest {
  status: TaskStatus
  position?: number
}

export interface FileInfo {
  path: string
  name: string
  is_directory: boolean
  size: number
  modified_at: string
  children?: FileInfo[]
}

export interface FileChangeEvent {
  type: 'created' | 'modified' | 'deleted' | 'renamed'
  path: string
  old_path?: string
  timestamp: string
  version?: number
}

export interface WriteFileRequest {
  path: string
  content: string
}

export interface CreateDirectoryRequest {
  path: string
}

export interface ExecuteTaskResponse {
  session_id: string
  status: string
}

export interface TaskExecutionState {
  isExecuting: boolean
  sessionId: string | null
  error: string | null
}

export type SessionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface Session {
  id: string
  task_id: string
  project_id: string
  status: SessionStatus
  prompt?: string
  output?: string
  error?: string
  started_at?: string
  completed_at?: string
  duration_ms: number
  created_at: string
  updated_at: string
  deleted_at?: string
}

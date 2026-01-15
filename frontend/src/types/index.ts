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

export interface Task {
  id: string
  project_id: string
  title: string
  description?: string
  status: 'todo' | 'in_progress' | 'ai_review' | 'human_review' | 'done'
  current_session_id?: string
  opencode_output?: string
  execution_duration_ms: number
  file_references?: Record<string, unknown>
  created_by: string
  created_at: string
  updated_at: string
}

export interface OpenCodeConfig {
  id: string
  project_id: string
  model: string
  provider: string
  provider_api_key_encrypted?: string
  tools?: Record<string, unknown>
  instructions?: string
  temperature: number
  max_tokens: number
  is_active: boolean
  version: number
  created_at: string
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

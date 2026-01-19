import axios from 'axios'

import type {
  CreateConfigRequest,
  CreateProjectRequest,
  CreateTaskRequest,
  ExecuteTaskResponse,
  FileInfo,
  MoveTaskRequest,
  OpenCodeConfig,
  Project,
  Session,
  Task,
  UpdateProjectRequest,
  UpdateTaskRequest,
  WriteFileRequest,
} from '@/types'

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export const api = axios.create({
  baseURL: `${API_BASE_URL}/api`,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
})

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export async function createProject(data: CreateProjectRequest): Promise<Project> {
  const response = await api.post<Project>('/projects', data)
  return response.data
}

export async function getProjects(): Promise<Project[]> {
  const response = await api.get<Project[]>('/projects')
  return response.data
}

export async function getProject(id: string): Promise<Project> {
  const response = await api.get<Project>(`/projects/${id}`)
  return response.data
}

export async function updateProject(id: string, data: UpdateProjectRequest): Promise<Project> {
  const response = await api.patch<Project>(`/projects/${id}`, data)
  return response.data
}

export async function deleteProject(id: string): Promise<void> {
  await api.delete(`/projects/${id}`)
}

export async function listTasks(projectId: string): Promise<Task[]> {
  const response = await api.get<Task[]>(`/projects/${projectId}/tasks`)
  return response.data
}

export async function createTask(projectId: string, data: CreateTaskRequest): Promise<Task> {
  const response = await api.post<Task>(`/projects/${projectId}/tasks`, data)
  return response.data
}

export async function getTask(projectId: string, taskId: string): Promise<Task> {
  const response = await api.get<Task>(`/projects/${projectId}/tasks/${taskId}`)
  return response.data
}

export async function updateTask(
  projectId: string,
  taskId: string,
  data: UpdateTaskRequest
): Promise<Task> {
  const response = await api.patch<Task>(`/projects/${projectId}/tasks/${taskId}`, data)
  return response.data
}

export async function moveTask(
  projectId: string,
  taskId: string,
  data: MoveTaskRequest
): Promise<Task> {
  const response = await api.patch<Task>(`/projects/${projectId}/tasks/${taskId}/move`, data)
  return response.data
}

export async function deleteTask(projectId: string, taskId: string): Promise<void> {
  await api.delete(`/projects/${projectId}/tasks/${taskId}`)
}

export async function executeTask(
  projectId: string,
  taskId: string
): Promise<ExecuteTaskResponse> {
  const response = await api.post<ExecuteTaskResponse>(
    `/projects/${projectId}/tasks/${taskId}/execute`
  )
  return response.data
}

export async function stopTaskExecution(projectId: string, taskId: string): Promise<void> {
  await api.post(`/projects/${projectId}/tasks/${taskId}/stop`)
}

export async function getTaskSessions(projectId: string, taskId: string): Promise<Session[]> {
  const response = await api.get<Session[]>(`/projects/${projectId}/tasks/${taskId}/sessions`)
  return response.data
}

export async function getFileTree(projectId: string, includeHidden = false): Promise<FileInfo> {
  const response = await api.get<FileInfo>(`/projects/${projectId}/files/tree`, {
    params: { include_hidden: includeHidden },
  })
  return response.data
}

export async function getFileContent(projectId: string, path: string): Promise<string> {
  const response = await api.get<{ content: string }>(`/projects/${projectId}/files/content`, {
    params: { path },
  })
  return response.data.content
}

export async function getFileInfo(projectId: string, path: string): Promise<FileInfo> {
  const response = await api.get<FileInfo>(`/projects/${projectId}/files/info`, {
    params: { path },
  })
  return response.data
}

export async function writeFile(projectId: string, data: WriteFileRequest): Promise<void> {
  await api.post(`/projects/${projectId}/files/write`, data)
}

export async function deleteFile(projectId: string, path: string): Promise<void> {
  await api.delete(`/projects/${projectId}/files`, {
    params: { path },
  })
}

export async function createDirectory(projectId: string, path: string): Promise<void> {
  await api.post(`/projects/${projectId}/files/mkdir`, { path })
}

export async function getActiveConfig(projectId: string): Promise<OpenCodeConfig> {
  const response = await api.get<OpenCodeConfig>(`/projects/${projectId}/config`)
  return response.data
}

export async function createOrUpdateConfig(
  projectId: string,
  data: CreateConfigRequest
): Promise<OpenCodeConfig> {
  const response = await api.post<OpenCodeConfig>(`/projects/${projectId}/config`, data)
  return response.data
}

export async function getConfigHistory(projectId: string): Promise<OpenCodeConfig[]> {
  const response = await api.get<OpenCodeConfig[]>(`/projects/${projectId}/config/versions`)
  return response.data
}

export async function rollbackConfig(projectId: string, version: number): Promise<void> {
  await api.post(`/projects/${projectId}/config/rollback/${version}`)
}

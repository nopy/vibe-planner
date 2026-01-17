import axios from 'axios'

import type {
  CreateProjectRequest,
  Project,
  UpdateProjectRequest,
} from '@/types'

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export const api = axios.create({
  baseURL: `${API_BASE_URL}/api`,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export async function createProject(
  data: CreateProjectRequest
): Promise<Project> {
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

export async function updateProject(
  id: string,
  data: UpdateProjectRequest
): Promise<Project> {
  const response = await api.patch<Project>(`/projects/${id}`, data)
  return response.data
}

export async function deleteProject(id: string): Promise<void> {
  await api.delete(`/projects/${id}`)
}

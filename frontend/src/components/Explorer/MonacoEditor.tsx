import { useState, useEffect, useRef, useCallback } from 'react'

import Editor, { OnMount } from '@monaco-editor/react'
import type { editor } from 'monaco-editor'

import { getFileContent } from '@/services/api'
import type { FileInfo } from '@/types'

interface MonacoEditorProps {
  projectId: string
  file: FileInfo | null
  onSave: (path: string, content: string) => Promise<void>
  onClose: () => void
  onDirtyChange: (path: string, isDirty: boolean) => void
}

const LANGUAGE_MAP: Record<string, string> = {
  ts: 'typescript',
  tsx: 'typescriptreact',
  js: 'javascript',
  jsx: 'javascriptreact',
  go: 'go',
  json: 'json',
  yaml: 'yaml',
  yml: 'yaml',
  md: 'markdown',
  css: 'css',
  html: 'html',
  sh: 'shell',
  bash: 'shell',
  py: 'python',
  sql: 'sql',
  dockerfile: 'dockerfile',
}

export function MonacoEditor({
  projectId,
  file,
  onSave,
  onClose,
  onDirtyChange,
}: MonacoEditorProps) {
  const [content, setContent] = useState<string>('')
  const [isLoading, setIsLoading] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isDirty, setIsDirty] = useState(false)

  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null)
  const saveTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const getLanguage = (filename: string): string => {
    const ext = filename.split('.').pop()?.toLowerCase() || ''
    return LANGUAGE_MAP[ext] || 'plaintext'
  }

  const loadContent = useCallback(async () => {
    if (!file) {
      setContent('')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const data = await getFileContent(projectId, file.path)
      setContent(data)
      setIsDirty(false)
      onDirtyChange(file.path, false)
    } catch (err) {
      console.error('Failed to load file:', err)
      setError('Failed to load file content')
    } finally {
      setIsLoading(false)
    }
  }, [projectId, file, onDirtyChange])

  useEffect(() => {
    loadContent()
  }, [loadContent])

  useEffect(() => {
    if (saveSuccess) {
      const timer = setTimeout(() => setSaveSuccess(false), 2000)
      return () => clearTimeout(timer)
    }
  }, [saveSuccess])

  const handleSave = useCallback(async (currentContent: string) => {
    if (!file) return

    setIsSaving(true)
    setError(null)

    try {
      await onSave(file.path, currentContent)
      setIsDirty(false)
      onDirtyChange(file.path, false)
      setSaveSuccess(true)
    } catch (err) {
      console.error('Failed to save:', err)
      setError('Failed to save changes')
    } finally {
      setIsSaving(false)
    }
  }, [file, onSave, onDirtyChange])

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        if (file && isDirty) {
          handleSave(editorRef.current?.getValue() || '')
        }
      }
    },
    [file, isDirty, handleSave]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  const handleEditorChange = (value: string | undefined) => {
    if (value === undefined || !file) return

    if (!isDirty) {
      setIsDirty(true)
      onDirtyChange(file.path, true)
    }

    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current)
    }
  }

  const handleEditorDidMount: OnMount = (editor) => {
    editorRef.current = editor

    editor.onDidBlurEditorText(() => {
      if (isDirty) {
        if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current)
        saveTimeoutRef.current = setTimeout(() => {
          handleSave(editor.getValue())
        }, 500)
      }
    })
  }

  if (!file) {
    return (
      <div className="h-full flex items-center justify-center bg-[#1e1e1e] text-gray-400">
        <p>Select a file to edit</p>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col bg-[#1e1e1e] relative">
      <div className="absolute top-2 right-4 z-10 flex items-center gap-3">
        {isSaving && <span className="text-yellow-400 text-xs animate-pulse">Saving...</span>}
        {saveSuccess && <span className="text-green-400 text-xs">Saved</span>}
        {isDirty && !isSaving && !saveSuccess && (
          <span className="text-gray-400 text-xs">Unsaved</span>
        )}
        <button 
          onClick={onClose}
          className="text-gray-500 hover:text-gray-300 ml-2"
          title="Close Editor"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {isLoading ? (
        <div className="h-full flex items-center justify-center text-gray-400">
          <div className="flex flex-col items-center gap-2">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
            <span>Loading...</span>
          </div>
        </div>
      ) : error ? (
        <div className="h-full flex items-center justify-center">
          <div className="bg-red-900/20 p-6 rounded-lg border border-red-800 text-center max-w-md">
            <h3 className="text-red-500 font-medium mb-2">Error</h3>
            <p className="text-gray-400 text-sm mb-4">{error}</p>
            <button
              onClick={loadContent}
              className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded text-sm transition-colors"
            >
              Retry
            </button>
          </div>
        </div>
      ) : (
        <Editor
          height="100%"
          defaultLanguage="plaintext"
          language={getLanguage(file.name)}
          value={content}
          theme="vs-dark"
          onChange={handleEditorChange}
          onMount={handleEditorDidMount}
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            automaticLayout: true,
            tabSize: 2,
            wordWrap: 'on',
          }}
        />
      )}
    </div>
  )
}

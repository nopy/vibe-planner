import { useEffect, useState, useCallback } from 'react'

import { CreateDirectoryModal } from './CreateDirectoryModal'
import { EditorTabs } from './EditorTabs'
import { FileTree } from './FileTree'
import { MonacoEditor } from './MonacoEditor'
import { RenameModal } from './RenameModal'
import { getFileTree, writeFile } from '@/services/api'
import type { FileInfo } from '@/types'

interface FileExplorerProps {
  projectId: string
}

export function FileExplorer({ projectId }: FileExplorerProps) {
  const [tree, setTree] = useState<FileInfo | null>(null)
  const [selectedNode, setSelectedNode] = useState<FileInfo | null>(null)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showHidden, setShowHidden] = useState(false)

  const [openFiles, setOpenFiles] = useState<Array<{ path: string; isDirty: boolean }>>([])
  const [activeFile, setActiveFile] = useState<string | null>(null)

  const [showCreateDirModal, setShowCreateDirModal] = useState(false)
  const [showRenameModal, setShowRenameModal] = useState(false)
  const [renameTarget, setRenameTarget] = useState<string | null>(null)

  const fetchTree = useCallback(async () => {
    setIsLoading(true)
    setError(null)

    try {
      const data = await getFileTree(projectId, showHidden)
      setTree(data)
      setExpandedNodes(prev => new Set(prev).add(data.path))
    } catch (err) {
      console.error('Failed to fetch file tree:', err)
      setError('Failed to load files. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }, [projectId, showHidden])

  useEffect(() => {
    fetchTree()
  }, [fetchTree])

  const handleSelect = (node: FileInfo) => {
    setSelectedNode(node)

    if (!node.is_directory) {
      if (!openFiles.some(f => f.path === node.path)) {
        setOpenFiles(prev => [...prev, { path: node.path, isDirty: false }])
      }
      setActiveFile(node.path)
    } else {
      handleToggleExpand(node.path)
    }
  }

  const handleToggleExpand = (path: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }

  const handleTabClick = (path: string) => {
    setActiveFile(path)
    const node = findNodeByPath(tree, path)
    if (node) setSelectedNode(node)
  }

  const handleTabClose = (path: string) => {
    const file = openFiles.find(f => f.path === path)
    if (file?.isDirty) {
      if (!window.confirm('You have unsaved changes. Close anyway?')) {
        return
      }
    }

    const newOpenFiles = openFiles.filter(f => f.path !== path)
    setOpenFiles(newOpenFiles)

    if (activeFile === path) {
      if (newOpenFiles.length > 0) {
        setActiveFile(newOpenFiles[newOpenFiles.length - 1].path)
      } else {
        setActiveFile(null)
      }
    }
  }

  const handleDirtyChange = (path: string, isDirty: boolean) => {
    setOpenFiles(prev => prev.map(f => (f.path === path ? { ...f, isDirty } : f)))
  }

  const handleSave = async (path: string, content: string) => {
    await writeFile(projectId, { path, content })
  }

  const handleRenameClick = () => {
    if (selectedNode) {
      setRenameTarget(selectedNode.path)
      setShowRenameModal(true)
    }
  }

  const findNodeByPath = (node: FileInfo | null, path: string): FileInfo | null => {
    if (!node) return null
    if (node.path === path) return node
    if (node.children) {
      for (const child of node.children) {
        const found = findNodeByPath(child, path)
        if (found) return found
      }
    }
    return null
  }

  const getActiveFileInfo = (): FileInfo | null => {
    if (!activeFile) return null
    return findNodeByPath(tree, activeFile) || { 
      path: activeFile, 
      name: activeFile.split('/').pop() || '', 
      is_directory: false, 
      size: 0, 
      modified_at: new Date().toISOString(),
      children: undefined 
    }
  }

  if (isLoading && !tree) {
    return (
      <div className="flex items-center justify-center h-full min-h-[400px]">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <p className="mt-2 text-sm text-gray-500">Loading files...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-4">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-600 text-sm">{error}</p>
          <button
            onClick={fetchTree}
            className="mt-2 px-3 py-1 bg-red-600 text-white text-xs rounded hover:bg-red-700"
          >
            Retry
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col md:flex-row border border-gray-200 rounded-lg overflow-hidden bg-white">
      <div className="w-full md:w-[30%] min-w-[250px] border-b md:border-b-0 md:border-r border-gray-200 flex flex-col h-[500px] md:h-[calc(100vh-200px)]">
        <div className="p-2 border-b border-gray-100 flex flex-col sm:flex-row sm:items-center justify-between gap-2 bg-gray-50">
          <div className="flex items-center space-x-2">
            <input
              type="checkbox"
              id="showHidden"
              checked={showHidden}
              onChange={e => setShowHidden(e.target.checked)}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500 h-4 w-4"
            />
            <label
              htmlFor="showHidden"
              className="text-xs text-gray-600 select-none cursor-pointer"
            >
              Show hidden
            </label>
          </div>
          <div className="flex gap-1">
            {selectedNode && (
              <button
                onClick={handleRenameClick}
                className="px-2 py-1 bg-white border border-gray-300 rounded text-xs text-gray-700 hover:bg-gray-50"
                title="Rename selected"
              >
                Rename
              </button>
            )}
            <button
              onClick={() => setShowCreateDirModal(true)}
              className="px-2 py-1 bg-white border border-gray-300 rounded text-xs text-gray-700 hover:bg-gray-50 flex items-center justify-center"
            >
              <span className="mr-1">+</span> Folder
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto overflow-x-hidden py-2">
          {!tree || (tree.children && tree.children.length === 0) ? (
            <div className="p-8 text-center">
              <span className="text-2xl block mb-2">📁</span>
              <p className="text-sm text-gray-500">No files found.</p>
              <p className="text-xs text-gray-400 mt-1">Create a folder to get started.</p>
            </div>
          ) : (
            <FileTree
              nodes={tree.children || []}
              selectedPath={selectedNode?.path || null}
              expandedNodes={expandedNodes}
              onSelect={handleSelect}
              onToggleExpand={handleToggleExpand}
            />
          )}
        </div>
      </div>

      <div className="w-full md:w-[70%] bg-[#1e1e1e] flex flex-col h-[500px] md:h-[calc(100vh-200px)]">
        <EditorTabs
          openFiles={openFiles}
          activeFile={activeFile}
          onTabClick={handleTabClick}
          onTabClose={handleTabClose}
        />
        <div className="flex-1 overflow-hidden relative">
          {activeFile ? (
            <MonacoEditor
              projectId={projectId}
              file={getActiveFileInfo()}
              onSave={handleSave}
              onClose={() => handleTabClose(activeFile)}
              onDirtyChange={handleDirtyChange}
            />
          ) : (
            <div className="h-full flex items-center justify-center text-gray-500 bg-gray-50">
              <div className="text-center">
                <svg
                  className="mx-auto h-12 w-12 mb-4 opacity-30"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1}
                    d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
                  />
                </svg>
                <p>Select a file to view</p>
              </div>
            </div>
          )}
        </div>
      </div>

      <CreateDirectoryModal
        projectId={projectId}
        isOpen={showCreateDirModal}
        onClose={() => setShowCreateDirModal(false)}
        onDirectoryCreated={() => {
          fetchTree()
        }}
        parentPath={selectedNode?.is_directory ? selectedNode.path : null}
      />

      <RenameModal
        isOpen={showRenameModal}
        onClose={() => {
          setShowRenameModal(false)
          setRenameTarget(null)
        }}
        onRenamed={() => {
          fetchTree()
        }}
        currentPath={renameTarget}
      />
    </div>
  )
}

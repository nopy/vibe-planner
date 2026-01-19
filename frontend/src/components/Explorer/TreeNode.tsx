import React from 'react'

import type { FileInfo } from '@/types'

interface TreeNodeProps {
  node: FileInfo
  depth: number
  isSelected: boolean
  isExpanded: boolean
  onSelect: () => void
  onToggleExpand?: () => void
  onContextMenu?: (event: React.MouseEvent) => void
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function TreeNode({
  node,
  depth,
  isSelected,
  isExpanded,
  onSelect,
  onToggleExpand,
  onContextMenu,
}: TreeNodeProps) {
  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (node.is_directory) {
      onToggleExpand?.()
    } else {
      onSelect()
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      e.stopPropagation()
      if (node.is_directory) {
        onToggleExpand?.()
      } else {
        onSelect()
      }
    } else if (e.key === 'ArrowRight') {
      e.preventDefault()
      e.stopPropagation()
      if (node.is_directory && !isExpanded) {
        onToggleExpand?.()
      }
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault()
      e.stopPropagation()
      if (node.is_directory && isExpanded) {
        onToggleExpand?.()
      }
    }
  }

  return (
    <div
      role="treeitem"
      aria-selected={isSelected}
      aria-expanded={node.is_directory ? isExpanded : undefined}
      tabIndex={0}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      onContextMenu={onContextMenu}
      className={`
        group flex items-center py-1 pr-2 cursor-pointer select-none outline-none transition-colors duration-200
        ${isSelected ? 'bg-blue-50 border-l-4 border-blue-500' : 'border-l-4 border-transparent hover:bg-gray-100'}
      `}
      style={{ paddingLeft: `${depth * 16 + 4}px` }}
    >
      <div className="flex-shrink-0 mr-2 w-4 text-center text-gray-400">
        {node.is_directory && <span className="text-xs">{isExpanded ? '▼' : '▶'}</span>}
      </div>

      <div className="flex-shrink-0 mr-2 text-xl">
        {node.is_directory ? (isExpanded ? '📂' : '📁') : '📄'}
      </div>

      <div className="flex-1 min-w-0 truncate">
        <span className={`text-sm ${isSelected ? 'font-medium text-blue-900' : 'text-gray-700'}`}>
          {node.name}
        </span>
      </div>

      {!node.is_directory && node.size > 0 && (
        <div className="flex-shrink-0 ml-2 text-xs text-gray-400">{formatFileSize(node.size)}</div>
      )}
    </div>
  )
}

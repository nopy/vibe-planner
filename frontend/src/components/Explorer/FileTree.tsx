import React from 'react'

import { TreeNode } from './TreeNode'
import type { FileInfo } from '@/types'

interface FileTreeProps {
  nodes: FileInfo[]
  selectedPath: string | null
  expandedNodes: Set<string>
  onSelect: (node: FileInfo) => void
  onToggleExpand: (path: string) => void
  onContextMenu?: (node: FileInfo, event: React.MouseEvent) => void
  depth?: number
}

export function FileTree({
  nodes,
  selectedPath,
  expandedNodes,
  onSelect,
  onToggleExpand,
  onContextMenu,
  depth = 0,
}: FileTreeProps) {
  const sortedNodes = [...nodes].sort((a, b) => {
    if (a.is_directory === b.is_directory) {
      return a.name.localeCompare(b.name)
    }
    return a.is_directory ? -1 : 1
  })

  return (
    <ul className="w-full" role={depth === 0 ? 'tree' : 'group'}>
      {sortedNodes.map(node => {
        const isExpanded = expandedNodes.has(node.path)
        const isSelected = selectedPath === node.path

        return (
          <li key={node.path} className="block">
            <TreeNode
              node={node}
              depth={depth}
              isSelected={isSelected}
              isExpanded={isExpanded}
              onSelect={() => onSelect(node)}
              onToggleExpand={() => onToggleExpand(node.path)}
              onContextMenu={e => onContextMenu?.(node, e)}
            />
            {node.is_directory && isExpanded && node.children && (
              <FileTree
                nodes={node.children}
                selectedPath={selectedPath}
                expandedNodes={expandedNodes}
                onSelect={onSelect}
                onToggleExpand={onToggleExpand}
                onContextMenu={onContextMenu}
                depth={depth + 1}
              />
            )}
          </li>
        )
      })}
    </ul>
  )
}

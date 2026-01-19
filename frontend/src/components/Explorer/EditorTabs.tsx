interface EditorTabsProps {
  openFiles: Array<{ path: string; isDirty: boolean }>
  activeFile: string | null
  onTabClick: (path: string) => void
  onTabClose: (path: string) => void
}

export function EditorTabs({ openFiles, activeFile, onTabClick, onTabClose }: EditorTabsProps) {
  if (openFiles.length === 0) {
    return (
      <div className="h-10 bg-gray-100 flex items-center justify-center border-b border-gray-200">
        <span className="text-sm text-gray-500">No files open</span>
      </div>
    )
  }

  return (
    <div className="flex bg-gray-100 border-b border-gray-200 overflow-x-auto whitespace-nowrap hide-scrollbar">
      {openFiles.map(file => {
        const isActive = file.path === activeFile
        const fileName = file.path.split('/').pop() || file.path

        return (
          <div
            key={file.path}
            className={`
              flex items-center gap-2 px-4 py-2 text-sm cursor-pointer select-none border-r border-gray-200 min-w-[120px] max-w-[200px]
              ${isActive ? 'bg-white text-blue-600 font-medium border-b-2 border-b-blue-600' : 'bg-gray-50 text-gray-600 hover:bg-gray-100 border-b-2 border-b-transparent'}
            `}
            onClick={() => onTabClick(file.path)}
          >
            <div className="flex-1 truncate flex items-center gap-1.5">
              {file.isDirty && <div className="w-2 h-2 rounded-full bg-blue-500 flex-shrink-0" />}
              <span className="truncate">{fileName}</span>
            </div>
            <button
              onClick={e => {
                e.stopPropagation()
                onTabClose(file.path)
              }}
              className="p-0.5 rounded hover:bg-gray-200 text-gray-400 hover:text-gray-600 transition-colors"
            >
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        )
      })}
    </div>
  )
}

import { useParams } from 'react-router-dom'

import { FileExplorer } from '@/components/Explorer/FileExplorer'

export function FileExplorerPage() {
  const { id } = useParams<{ id: string }>()

  if (!id) {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center">
        <p className="text-red-600">Project ID is missing</p>
      </div>
    )
  }

  return <FileExplorer projectId={id} />
}

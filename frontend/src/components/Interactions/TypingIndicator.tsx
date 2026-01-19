export function TypingIndicator() {
  return (
    <div className="flex justify-start px-4 pb-4">
      <div className="max-w-[70%] rounded-lg border border-gray-200 bg-white px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0 rounded-full bg-purple-100 px-2 py-0.5 text-xs font-medium text-purple-700">
            AI
          </div>
        </div>
        <div className="mt-1.5 flex items-center gap-1">
          <div className="h-2 w-2 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.3s]" />
          <div className="h-2 w-2 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.15s]" />
          <div className="h-2 w-2 animate-bounce rounded-full bg-gray-400" />
        </div>
      </div>
    </div>
  )
}

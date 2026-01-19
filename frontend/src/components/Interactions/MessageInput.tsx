import { useState, useRef, KeyboardEvent } from 'react'

interface MessageInputProps {
  onSend: (content: string) => void
  disabled: boolean
}

const MAX_CHARS = 2000

export function MessageInput({ onSend, disabled }: MessageInputProps) {
  const [message, setMessage] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const handleSend = () => {
    const trimmed = message.trim()
    if (!trimmed || disabled) return

    onSend(trimmed)
    setMessage('')

    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
    }
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleInput = () => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
      textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`
    }
  }

  const charCount = message.length
  const isOverLimit = charCount > MAX_CHARS
  const canSend = message.trim().length > 0 && !disabled && !isOverLimit

  return (
    <div className="p-4">
      <div className="relative">
        <textarea
          ref={textareaRef}
          value={message}
          onChange={e => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          onInput={handleInput}
          placeholder={disabled ? 'Disconnected - cannot send messages' : 'Type a message... (Shift+Enter for new line)'}
          disabled={disabled}
          rows={1}
          className={`w-full resize-none rounded-lg border px-4 py-3 pr-28 text-sm focus:outline-none focus:ring-2 ${
            disabled
              ? 'border-gray-200 bg-gray-50 text-gray-400 cursor-not-allowed'
              : isOverLimit
                ? 'border-red-300 bg-white focus:border-red-500 focus:ring-red-500'
                : 'border-gray-300 bg-white focus:border-blue-500 focus:ring-blue-500'
          }`}
          style={{ minHeight: '44px', maxHeight: '200px' }}
        />
        <div className="absolute bottom-3 right-3 flex items-center gap-2">
          <span
            className={`text-xs ${
              isOverLimit ? 'text-red-600 font-medium' : 'text-gray-400'
            }`}
          >
            {charCount}/{MAX_CHARS}
          </span>
          <button
            onClick={handleSend}
            disabled={!canSend}
            className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors ${
              canSend
                ? 'bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800'
                : 'bg-gray-200 text-gray-400 cursor-not-allowed'
            }`}
          >
            Send
          </button>
        </div>
      </div>
      {isOverLimit && (
        <p className="mt-1 text-xs text-red-600">
          Message exceeds maximum length of {MAX_CHARS} characters
        </p>
      )}
    </div>
  )
}

import { useEffect, useRef } from 'react'

import { MessageInput } from './MessageInput'
import { MessageList } from './MessageList'
import { TypingIndicator } from './TypingIndicator'
import { useInteractions } from '@/hooks/useInteractions'

interface InteractionPanelProps {
  projectId: string
  taskId: string
}

export function InteractionPanel({ projectId, taskId }: InteractionPanelProps) {
  const { messages, isConnected, isTyping, error, sendMessage, reconnect } = useInteractions(
    projectId,
    taskId
  )
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  return (
    <div className="flex h-full flex-col bg-gray-50">
      {error && (
        <div className="border-b border-red-200 bg-red-50 p-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="h-2 w-2 rounded-full bg-red-500" />
              <p className="text-sm text-red-700">{error}</p>
            </div>
            {!isConnected && (
              <button
                onClick={reconnect}
                className="rounded bg-red-600 px-3 py-1 text-sm text-white hover:bg-red-700"
              >
                Reconnect
              </button>
            )}
          </div>
        </div>
      )}

      <div className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-3">
        <div className="flex items-center gap-2">
          <h3 className="font-semibold text-gray-900">AI Interaction</h3>
          <div className="flex items-center gap-1.5">
            <div className={`h-2 w-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-gray-300'}`} />
            <span className="text-xs text-gray-500">{isConnected ? 'Connected' : 'Disconnected'}</span>
          </div>
        </div>
        {messages.length > 0 && (
          <span className="text-sm text-gray-500">{messages.length} message{messages.length !== 1 ? 's' : ''}</span>
        )}
      </div>

      <div className="flex-1 overflow-hidden">
        <MessageList messages={messages} isTyping={isTyping} />
        {isTyping && <TypingIndicator />}
        <div ref={messagesEndRef} />
      </div>

      <div className="border-t border-gray-200 bg-white">
        <MessageInput onSend={sendMessage} disabled={!isConnected} />
      </div>
    </div>
  )
}

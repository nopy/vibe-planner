import type { Interaction } from '@/types'

interface MessageListProps {
  messages: Interaction[]
  isTyping: boolean
}

export function MessageList({ messages, isTyping }: MessageListProps) {
  if (messages.length === 0 && !isTyping) {
    return (
      <div className="flex h-full items-center justify-center p-8">
        <div className="text-center">
          <div className="mb-3 text-4xl">💬</div>
          <p className="mb-2 font-medium text-gray-700">No messages yet</p>
          <p className="text-sm text-gray-500">Start a conversation with the AI agent</p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto p-4 space-y-4">
      {messages.map((message, index) => {
        const isUser = message.message_type === 'user_message'
        const isAgent = message.message_type === 'agent_response'
        const isSystem = message.message_type === 'system_notification'

        if (isSystem) {
          return (
            <div key={index} className="flex justify-center">
              <div className="max-w-md rounded-lg bg-gray-100 px-4 py-2 text-center">
                <p className="text-sm italic text-gray-600">{message.content}</p>
                <p className="mt-1 text-xs text-gray-400">
                  {new Date(message.created_at).toLocaleTimeString()}
                </p>
              </div>
            </div>
          )
        }

        return (
          <div key={index} className={`flex ${isUser ? 'justify-end' : 'justify-start'}`}>
            <div
              className={`max-w-[70%] rounded-lg px-4 py-3 ${
                isUser
                  ? 'bg-blue-600 text-white'
                  : 'bg-white border border-gray-200 text-gray-900'
              }`}
            >
              <div className="flex items-start gap-2">
                {isAgent && (
                  <div className="flex-shrink-0 rounded-full bg-purple-100 px-2 py-0.5 text-xs font-medium text-purple-700">
                    AI
                  </div>
                )}
                {isUser && (
                  <div className="flex-shrink-0 rounded-full bg-blue-800 px-2 py-0.5 text-xs font-medium text-blue-100">
                    You
                  </div>
                )}
              </div>
              <p className="mt-1.5 whitespace-pre-wrap break-words text-sm">{message.content}</p>
              <p
                className={`mt-2 text-xs ${isUser ? 'text-blue-200' : 'text-gray-400'}`}
              >
                {new Date(message.created_at).toLocaleTimeString()}
              </p>
            </div>
          </div>
        )
      })}
    </div>
  )
}

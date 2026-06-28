import { contentToText, type ChatMessage } from '../lib/parse'

/** Render a chat thread as role/content bubbles. */
export function MessageThread({
  system,
  messages,
}: {
  system?: string
  messages: ChatMessage[]
}) {
  const hasSystem = system && system.trim().length > 0
  if (!hasSystem && messages.length === 0) {
    return <div className="dim">No messages.</div>
  }
  return (
    <div className="chat-thread">
      {hasSystem && <Bubble role="system" text={system!} />}
      {messages.map((m, i) => (
        <Bubble key={i} role={m.role} text={contentToText(m.content)} />
      ))}
    </div>
  )
}

function roleClass(role: string): string {
  const r = role.toLowerCase()
  if (r === 'system') return 'role-system'
  if (r === 'user' || r === 'human') return 'role-user'
  if (r === 'assistant' || r === 'model' || r === 'ai') return 'role-assistant'
  if (r === 'tool') return 'role-tool'
  return 'role-other'
}

export function Bubble({ role, text }: { role: string; text: string }) {
  return (
    <div className={`bubble ${roleClass(role)}`}>
      <div className="bubble-role">{role}</div>
      <div className="bubble-content">{text || <span className="dim">(empty)</span>}</div>
    </div>
  )
}

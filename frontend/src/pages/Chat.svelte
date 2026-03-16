<script>
  import { onMount, onDestroy } from 'svelte'
  import { marked } from 'marked'

  // Configure marked for dark code blocks
  marked.setOptions({
    breaks: true,
    gfm: true,
  })

  // Wails runtime bindings
  const wails = window['go']?.['main']?.['App']
  const runtime = window.runtime

  // Session state
  let sessions = []
  let activeSessionKey = 'general'

  // Chat state
  let messages = []
  let inputText = ''
  let isStreaming = false
  let streamingContent = ''
  let messagesContainer

  // Image attachment state
  let attachedImages = []
  let fileInput

  // New session modal
  let showNewSession = false
  let newSessionName = ''
  let newSessionPrompt = ''

  // Load sessions and set up events
  onMount(async () => {
    await loadSessions()
    await switchSession(activeSessionKey)

    if (runtime) {
      runtime.EventsOn('chat:chunk', (chunk) => {
        streamingContent += chunk
        scrollToBottom()
      })
      runtime.EventsOn('chat:done', (fullText) => {
        messages = [...messages, { role: 'assistant', content: fullText }]
        streamingContent = ''
        isStreaming = false
        loadSessions() // refresh last message previews
        scrollToBottom()
      })
      runtime.EventsOn('chat:error', (error) => {
        messages = [...messages, { role: 'assistant', content: `**Error:** ${error}` }]
        streamingContent = ''
        isStreaming = false
        scrollToBottom()
      })
    }
  })

  onDestroy(() => {
    if (runtime) {
      runtime.EventsOff('chat:chunk')
      runtime.EventsOff('chat:done')
      runtime.EventsOff('chat:error')
    }
  })

  async function loadSessions() {
    if (wails) {
      try {
        const result = await wails.ListChatSessions()
        sessions = result || []
      } catch (e) {
        console.error('Failed to load sessions:', e)
      }
    }
  }

  async function switchSession(key) {
    activeSessionKey = key
    if (wails) {
      try {
        const history = await wails.SwitchSession(key)
        messages = history || []
        streamingContent = ''
        isStreaming = false
        await loadSessions() // refresh unread counts
        scrollToBottom()
      } catch (e) {
        console.error('Failed to switch session:', e)
      }
    }
  }

  async function createSession() {
    if (!newSessionName.trim()) return
    if (wails) {
      try {
        const session = await wails.CreateChatSession(newSessionName.trim(), newSessionPrompt.trim())
        showNewSession = false
        newSessionName = ''
        newSessionPrompt = ''
        await loadSessions()
        await switchSession(session.key)
      } catch (e) {
        console.error('Failed to create session:', e)
      }
    }
  }

  async function deleteSession(key) {
    if (key === 'general') return
    if (wails) {
      try {
        await wails.DeleteSession(key)
        if (activeSessionKey === key) {
          activeSessionKey = 'general'
          await switchSession('general')
        }
        await loadSessions()
      } catch (e) {
        console.error('Failed to delete session:', e)
      }
    }
  }

  async function sendMessage() {
    const text = inputText.trim()
    if (!text || isStreaming) return

    const userContent = text
    inputText = ''
    isStreaming = true
    streamingContent = ''

    // Show user message immediately
    if (attachedImages.length > 0) {
      messages = [...messages, { role: 'user', content: text, images: attachedImages.map(i => i.preview) }]
    } else {
      messages = [...messages, { role: 'user', content: text }]
    }
    scrollToBottom()

    if (wails) {
      try {
        if (attachedImages.length > 0) {
          const dataUrls = attachedImages.map(i => i.dataUrl)
          await wails.StreamChatWithImages(activeSessionKey, userContent, dataUrls)
          attachedImages = []
        } else {
          await wails.StreamChat(activeSessionKey, userContent)
        }
      } catch (e) {
        console.error('Failed to send:', e)
        isStreaming = false
      }
    }
  }

  async function clearChat() {
    if (wails) {
      await wails.ClearChatHistory(activeSessionKey)
      messages = []
      streamingContent = ''
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  function handleFileSelect(e) {
    const files = Array.from(e.target.files || [])
    for (const file of files) {
      if (!file.type.startsWith('image/')) continue
      const reader = new FileReader()
      reader.onload = (ev) => {
        attachedImages = [...attachedImages, {
          name: file.name,
          preview: ev.target.result,
          dataUrl: ev.target.result,
        }]
      }
      reader.readAsDataURL(file)
    }
    if (fileInput) fileInput.value = ''
  }

  function removeImage(index) {
    attachedImages = attachedImages.filter((_, i) => i !== index)
  }

  function scrollToBottom() {
    setTimeout(() => {
      if (messagesContainer) {
        messagesContainer.scrollTop = messagesContainer.scrollHeight
      }
    }, 10)
  }

  function renderMarkdown(text) {
    if (typeof text !== 'string') return ''
    return marked.parse(text)
  }

  function getMessageText(content) {
    if (typeof content === 'string') return content
    if (Array.isArray(content)) {
      const textPart = content.find(p => p.type === 'text')
      return textPart ? textPart.text : ''
    }
    return String(content)
  }

  function formatTime(isoString) {
    if (!isoString) return ''
    try {
      const d = new Date(isoString)
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    } catch {
      return ''
    }
  }

  $: activeSession = sessions.find(s => s.key === activeSessionKey)
</script>

<div class="chat-page">
  <!-- Session Sidebar -->
  <div class="session-panel">
    <div class="session-header">
      <span class="session-title">Channels</span>
      <button class="new-session-btn" on:click={() => showNewSession = true}>+</button>
    </div>
    <div class="session-list">
      {#each sessions as session}
        <button
          class="session-item"
          class:active={session.key === activeSessionKey}
          on:click={() => switchSession(session.key)}
        >
          <div class="session-item-top">
            <span class="session-icon">{session.icon}</span>
            <span class="session-name">{session.name}</span>
            {#if session.unread > 0}
              <span class="unread-badge">{session.unread}</span>
            {/if}
            {#if session.key !== 'general'}
              <button class="delete-session-btn" on:click|stopPropagation={() => deleteSession(session.key)}>x</button>
            {/if}
          </div>
          <div class="session-preview">{session.lastMessage || ''}</div>
          <div class="session-time">{formatTime(session.lastTime)}</div>
        </button>
      {/each}
    </div>
  </div>

  <!-- Chat Area -->
  <div class="chat-area">
    <div class="chat-header">
      <div class="chat-header-left">
        {#if activeSession}
          <span class="chat-header-icon">{activeSession.icon}</span>
          <span class="chat-header-name">{activeSession.name}</span>
        {/if}
      </div>
      <button class="clear-btn" on:click={clearChat}>Clear</button>
    </div>

    <div class="messages" bind:this={messagesContainer}>
      {#if messages.length === 0 && !isStreaming}
        <div class="empty-state">
          <div class="empty-icon">⬡</div>
          <div class="empty-text">Start a conversation</div>
        </div>
      {/if}

      {#each messages as msg}
        <div class="message-row" class:user={msg.role === 'user'} class:assistant={msg.role === 'assistant'}>
          <div class="message-bubble" class:user-bubble={msg.role === 'user'} class:assistant-bubble={msg.role === 'assistant'}>
            {#if msg.images && msg.images.length > 0}
              <div class="message-images">
                {#each msg.images as img}
                  <img src={img} alt="attached" class="message-image-thumb" />
                {/each}
              </div>
            {/if}
            <div class="message-content">{@html renderMarkdown(getMessageText(msg.content))}</div>
          </div>
        </div>
      {/each}

      {#if isStreaming && streamingContent}
        <div class="message-row assistant">
          <div class="message-bubble assistant-bubble">
            <div class="message-content">{@html renderMarkdown(streamingContent)}</div>
          </div>
        </div>
      {/if}

      {#if isStreaming && !streamingContent}
        <div class="message-row assistant">
          <div class="message-bubble assistant-bubble">
            <div class="typing-indicator">
              <span class="dot"></span>
              <span class="dot"></span>
              <span class="dot"></span>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- Image Previews -->
    {#if attachedImages.length > 0}
      <div class="image-previews">
        {#each attachedImages as img, i}
          <div class="image-preview-item">
            <img src={img.preview} alt={img.name} />
            <button class="remove-image" on:click={() => removeImage(i)}>x</button>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Input -->
    <div class="input-area">
      <button class="attach-btn" on:click={() => fileInput.click()}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48" />
        </svg>
      </button>
      <input type="file" accept="image/*" multiple bind:this={fileInput} on:change={handleFileSelect} style="display:none" />
      <textarea
        placeholder="Type a message..."
        bind:value={inputText}
        on:keydown={handleKeydown}
        disabled={isStreaming}
        rows="1"
      ></textarea>
      <button class="send-btn" on:click={sendMessage} disabled={isStreaming || !inputText.trim()}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="22" y1="2" x2="11" y2="13"></line>
          <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
        </svg>
      </button>
    </div>
  </div>

  <!-- New Session Modal -->
  {#if showNewSession}
    <div class="modal-overlay" on:click={() => showNewSession = false}>
      <div class="modal" on:click|stopPropagation>
        <h3>New Channel</h3>
        <label>
          Name
          <input type="text" bind:value={newSessionName} placeholder="e.g. Project Alpha" />
        </label>
        <label>
          System Prompt (optional)
          <textarea bind:value={newSessionPrompt} placeholder="Give this channel a personality or focus area..." rows="3"></textarea>
        </label>
        <div class="modal-actions">
          <button class="modal-cancel" on:click={() => showNewSession = false}>Cancel</button>
          <button class="modal-create" on:click={createSession}>Create</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .chat-page {
    display: flex;
    height: 100%;
    overflow: hidden;
    margin: -24px;
  }

  /* Session Panel */
  .session-panel {
    width: 250px;
    min-width: 250px;
    background: var(--bg-sidebar, #0a0a0a);
    border-right: 1px solid var(--border-subtle, #1a1a2e);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .session-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
    border-bottom: 1px solid var(--border-subtle, #1a1a2e);
  }

  .session-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-secondary, #888);
    text-transform: uppercase;
    letter-spacing: 1px;
  }

  .new-session-btn {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    border: 1px solid var(--border, #2a2a3e);
    background: transparent;
    color: var(--accent, #7c3aed);
    font-size: 16px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
  }

  .new-session-btn:hover {
    background: var(--accent-subtle, rgba(124, 58, 237, 0.1));
  }

  .session-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
  }

  .session-item {
    width: 100%;
    text-align: left;
    padding: 10px 12px;
    border-radius: 8px;
    border: none;
    background: transparent;
    cursor: pointer;
    transition: all 0.15s;
    margin-bottom: 2px;
    display: block;
  }

  .session-item:hover {
    background: var(--accent-subtle, rgba(124, 58, 237, 0.08));
  }

  .session-item.active {
    background: var(--accent-subtle, rgba(124, 58, 237, 0.15));
    border-left: 2px solid var(--accent, #7c3aed);
  }

  .session-item-top {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .session-icon {
    font-size: 14px;
  }

  .session-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary, #e0e0e0);
    flex: 1;
  }

  .unread-badge {
    background: var(--accent, #7c3aed);
    color: white;
    font-size: 10px;
    padding: 1px 6px;
    border-radius: 10px;
    font-weight: 600;
  }

  .delete-session-btn {
    background: transparent;
    border: none;
    color: var(--text-muted, #555);
    cursor: pointer;
    font-size: 12px;
    padding: 2px 4px;
    border-radius: 4px;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .session-item:hover .delete-session-btn {
    opacity: 1;
  }

  .delete-session-btn:hover {
    color: var(--error, #ef4444);
    background: rgba(239, 68, 68, 0.1);
  }

  .session-preview {
    font-size: 11px;
    color: var(--text-muted, #555);
    margin-top: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .session-time {
    font-size: 10px;
    color: var(--text-muted, #444);
    margin-top: 2px;
  }

  /* Chat Area */
  .chat-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--bg-primary, #0f0f0f);
  }

  .chat-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 20px;
    border-bottom: 1px solid var(--border-subtle, #1a1a2e);
    background: var(--bg-sidebar, #0a0a0a);
  }

  .chat-header-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .chat-header-icon {
    font-size: 18px;
  }

  .chat-header-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary, #e0e0e0);
  }

  .clear-btn {
    padding: 4px 12px;
    border-radius: 6px;
    border: 1px solid var(--border, #2a2a3e);
    background: transparent;
    color: var(--text-secondary, #888);
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .clear-btn:hover {
    background: rgba(239, 68, 68, 0.1);
    color: var(--error, #ef4444);
    border-color: rgba(239, 68, 68, 0.3);
  }

  /* Messages */
  .messages {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    color: var(--text-muted, #555);
  }

  .empty-icon {
    font-size: 48px;
    color: var(--accent, #7c3aed);
    opacity: 0.3;
  }

  .empty-text {
    font-size: 14px;
  }

  .message-row {
    display: flex;
  }

  .message-row.user {
    justify-content: flex-end;
  }

  .message-row.assistant {
    justify-content: flex-start;
  }

  .message-bubble {
    max-width: 75%;
    padding: 10px 14px;
    border-radius: 12px;
    word-wrap: break-word;
    overflow-wrap: break-word;
  }

  .user-bubble {
    background: #2a1f3d;
    border-bottom-right-radius: 4px;
  }

  .assistant-bubble {
    background: #1e1e1e;
    border-bottom-left-radius: 4px;
  }

  .message-content {
    font-size: 13px;
    line-height: 1.6;
    color: var(--text-primary, #e0e0e0);
  }

  .message-content :global(p) {
    margin: 0 0 8px 0;
  }

  .message-content :global(p:last-child) {
    margin-bottom: 0;
  }

  .message-content :global(pre) {
    background: #0d0d0d;
    padding: 12px;
    border-radius: 8px;
    overflow-x: auto;
    margin: 8px 0;
    border: 1px solid #2a2a3e;
  }

  .message-content :global(code) {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 12px;
  }

  .message-content :global(:not(pre) > code) {
    background: #2a2a3e;
    padding: 2px 6px;
    border-radius: 4px;
  }

  .message-content :global(ul), .message-content :global(ol) {
    margin: 4px 0;
    padding-left: 20px;
  }

  .message-content :global(a) {
    color: var(--accent, #7c3aed);
  }

  .message-images {
    display: flex;
    gap: 8px;
    margin-bottom: 8px;
    flex-wrap: wrap;
  }

  .message-image-thumb {
    width: 120px;
    height: 80px;
    object-fit: cover;
    border-radius: 8px;
    border: 1px solid #2a2a3e;
  }

  /* Typing indicator */
  .typing-indicator {
    display: flex;
    gap: 4px;
    padding: 4px 0;
  }

  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--accent, #7c3aed);
    animation: bounce 1.2s infinite;
  }

  .dot:nth-child(2) {
    animation-delay: 0.2s;
  }

  .dot:nth-child(3) {
    animation-delay: 0.4s;
  }

  @keyframes bounce {
    0%, 60%, 100% { transform: translateY(0); opacity: 0.4; }
    30% { transform: translateY(-6px); opacity: 1; }
  }

  /* Image previews */
  .image-previews {
    display: flex;
    gap: 8px;
    padding: 8px 20px;
    background: var(--bg-sidebar, #0a0a0a);
    border-top: 1px solid var(--border-subtle, #1a1a2e);
    flex-wrap: wrap;
  }

  .image-preview-item {
    position: relative;
    width: 60px;
    height: 60px;
  }

  .image-preview-item img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 8px;
    border: 1px solid #2a2a3e;
  }

  .remove-image {
    position: absolute;
    top: -4px;
    right: -4px;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: var(--error, #ef4444);
    color: white;
    border: none;
    font-size: 10px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  /* Input */
  .input-area {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    padding: 12px 20px;
    border-top: 1px solid var(--border-subtle, #1a1a2e);
    background: var(--bg-sidebar, #0a0a0a);
  }

  .attach-btn, .send-btn {
    width: 36px;
    height: 36px;
    border-radius: 8px;
    border: 1px solid var(--border, #2a2a3e);
    background: transparent;
    color: var(--text-secondary, #888);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: all 0.15s;
  }

  .attach-btn:hover {
    background: var(--accent-subtle, rgba(124, 58, 237, 0.1));
    color: var(--accent, #7c3aed);
  }

  .send-btn {
    background: var(--accent, #7c3aed);
    border-color: var(--accent, #7c3aed);
    color: white;
  }

  .send-btn:hover:not(:disabled) {
    opacity: 0.85;
  }

  .send-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  textarea {
    flex: 1;
    background: #1a1a2e;
    border: 1px solid var(--border, #2a2a3e);
    border-radius: 8px;
    padding: 8px 12px;
    color: var(--text-primary, #e0e0e0);
    font-size: 13px;
    font-family: inherit;
    resize: none;
    min-height: 36px;
    max-height: 120px;
    outline: none;
    transition: border-color 0.15s;
  }

  textarea:focus {
    border-color: var(--accent, #7c3aed);
  }

  textarea::placeholder {
    color: var(--text-muted, #555);
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: #1a1a2e;
    border: 1px solid var(--border, #2a2a3e);
    border-radius: 12px;
    padding: 24px;
    width: 400px;
    max-width: 90vw;
  }

  .modal h3 {
    margin: 0 0 16px 0;
    font-size: 16px;
    color: var(--text-primary, #e0e0e0);
  }

  .modal label {
    display: block;
    margin-bottom: 12px;
    font-size: 12px;
    color: var(--text-secondary, #888);
  }

  .modal input[type="text"],
  .modal textarea {
    display: block;
    width: 100%;
    margin-top: 6px;
    background: #0f0f0f;
    border: 1px solid var(--border, #2a2a3e);
    border-radius: 6px;
    padding: 8px 10px;
    color: var(--text-primary, #e0e0e0);
    font-size: 13px;
    font-family: inherit;
    outline: none;
    box-sizing: border-box;
  }

  .modal input:focus, .modal textarea:focus {
    border-color: var(--accent, #7c3aed);
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 16px;
  }

  .modal-cancel, .modal-create {
    padding: 6px 16px;
    border-radius: 6px;
    font-size: 13px;
    cursor: pointer;
    border: none;
  }

  .modal-cancel {
    background: transparent;
    border: 1px solid var(--border, #2a2a3e);
    color: var(--text-secondary, #888);
  }

  .modal-create {
    background: var(--accent, #7c3aed);
    color: white;
  }

  .modal-create:hover {
    opacity: 0.85;
  }

  /* Scrollbar */
  .messages::-webkit-scrollbar,
  .session-list::-webkit-scrollbar {
    width: 6px;
  }

  .messages::-webkit-scrollbar-track,
  .session-list::-webkit-scrollbar-track {
    background: transparent;
  }

  .messages::-webkit-scrollbar-thumb,
  .session-list::-webkit-scrollbar-thumb {
    background: #2a2a3e;
    border-radius: 3px;
  }
</style>

<script>
  import { onMount } from 'svelte'

  let emails = []
  let loading = true
  let error = ''
  let selectedEmail = null
  let selectedDetail = null
  let loadingDetail = false
  let showCompose = false
  let composeTo = ''
  let composeSubject = ''
  let composeBody = ''
  let sending = false
  let sendError = ''
  let sendSuccess = false

  onMount(() => {
    loadEmails()
  })

  async function loadEmails() {
    loading = true
    error = ''
    try {
      const result = await window['go']['main']['App']['GetEmails'](20)
      if (result.error) {
        error = result.error
      } else {
        emails = result.emails || []
      }
    } catch (e) {
      error = e.toString()
    }
    loading = false
  }

  async function openEmail(email) {
    selectedEmail = email
    loadingDetail = true
    selectedDetail = null
    try {
      const result = await window['go']['main']['App']['GetEmail'](email.id)
      if (result.error) {
        selectedDetail = { body: 'Error loading email: ' + result.error }
      } else {
        selectedDetail = result.email
      }
      // Mark as read if unread
      if (!email.isRead) {
        await window['go']['main']['App']['MarkAsRead'](email.id)
        email.isRead = true
        emails = emails
      }
    } catch (e) {
      selectedDetail = { body: 'Error: ' + e.toString() }
    }
    loadingDetail = false
  }

  function closeDetail() {
    selectedEmail = null
    selectedDetail = null
  }

  function openCompose() {
    composeTo = ''
    composeSubject = ''
    composeBody = ''
    sendError = ''
    sendSuccess = false
    showCompose = true
  }

  function closeCompose() {
    showCompose = false
  }

  async function sendEmail() {
    if (!composeTo || !composeSubject) {
      sendError = 'To and Subject are required'
      return
    }
    sending = true
    sendError = ''
    sendSuccess = false
    try {
      const result = await window['go']['main']['App']['SendEmail'](composeTo, composeSubject, composeBody)
      if (result.error) {
        sendError = result.error
      } else {
        sendSuccess = true
        setTimeout(() => {
          showCompose = false
          loadEmails()
        }, 1200)
      }
    } catch (e) {
      sendError = e.toString()
    }
    sending = false
  }

  function getUnreadCount() {
    return emails.filter(e => !e.isRead).length
  }
</script>

<div class="email-page">
  <div class="email-header">
    <div class="header-actions">
      <button class="btn btn-primary" on:click={openCompose}>Compose</button>
      <button class="btn btn-ghost" on:click={loadEmails} disabled={loading}>
        {#if loading}
          <span class="spinner"></span>
        {:else}
          Refresh
        {/if}
      </button>
    </div>
  </div>

  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  <div class="email-card">
    <div class="email-toolbar">
      <span class="toolbar-label">Inbox</span>
      {#if !loading}
        <span class="toolbar-count">{getUnreadCount()} unread</span>
      {/if}
    </div>

    {#if loading && emails.length === 0}
      <div class="loading-state">
        <span class="spinner"></span>
        <span>Loading emails...</span>
      </div>
    {:else if emails.length === 0}
      <div class="empty-state">No emails found</div>
    {:else}
      <div class="email-list">
        {#each emails as email}
          <div
            class="email-row"
            class:unread={!email.isRead}
            class:selected={selectedEmail && selectedEmail.id === email.id}
            on:click={() => openEmail(email)}
          >
            <div class="email-dot" class:visible={!email.isRead}></div>
            <div class="email-from">{email.from}</div>
            <div class="email-subject">
              <span class="subject-text">{email.subject}</span>
              <span class="email-snippet">{email.snippet}</span>
            </div>
            <div class="email-time">{email.date}</div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Email Detail Panel -->
  {#if selectedEmail}
    <div class="detail-overlay" on:click={closeDetail}>
      <div class="detail-panel" on:click|stopPropagation>
        <div class="detail-header">
          <div class="detail-subject">{selectedEmail.subject}</div>
          <button class="btn-close" on:click={closeDetail}>&times;</button>
        </div>
        <div class="detail-meta">
          <span class="detail-from">{selectedDetail?.from || selectedEmail.from}</span>
          {#if selectedDetail?.to}
            <span class="detail-to">To: {selectedDetail.to}</span>
          {/if}
          <span class="detail-date">{selectedDetail?.date || selectedEmail.date}</span>
        </div>
        <div class="detail-body">
          {#if loadingDetail}
            <div class="loading-state">
              <span class="spinner"></span>
              <span>Loading...</span>
            </div>
          {:else if selectedDetail}
            <pre class="body-text">{selectedDetail.body}</pre>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  <!-- Compose Modal -->
  {#if showCompose}
    <div class="detail-overlay" on:click={closeCompose}>
      <div class="compose-panel" on:click|stopPropagation>
        <div class="detail-header">
          <div class="detail-subject">New Email</div>
          <button class="btn-close" on:click={closeCompose}>&times;</button>
        </div>
        <div class="compose-form">
          <input
            class="compose-input"
            type="email"
            placeholder="To"
            bind:value={composeTo}
          />
          <input
            class="compose-input"
            type="text"
            placeholder="Subject"
            bind:value={composeSubject}
          />
          <textarea
            class="compose-textarea"
            placeholder="Write your message..."
            bind:value={composeBody}
            rows="10"
          ></textarea>
          {#if sendError}
            <div class="send-error">{sendError}</div>
          {/if}
          {#if sendSuccess}
            <div class="send-success">Email sent!</div>
          {/if}
          <div class="compose-actions">
            <button class="btn btn-primary" on:click={sendEmail} disabled={sending}>
              {#if sending}
                <span class="spinner"></span> Sending...
              {:else}
                Send
              {/if}
            </button>
            <button class="btn btn-ghost" on:click={closeCompose}>Cancel</button>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .email-page {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .email-header {
    display: flex;
    justify-content: flex-end;
    align-items: center;
  }

  .header-actions {
    display: flex;
    gap: 8px;
  }

  .btn {
    padding: 8px 16px;
    border-radius: var(--radius-md);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: all var(--transition-fast);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .btn-primary {
    background: var(--accent);
    color: white;
  }

  .btn-primary:hover {
    opacity: 0.9;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-ghost {
    background: transparent;
    color: var(--text-secondary);
    border: 1px solid var(--border-subtle);
  }

  .btn-ghost:hover {
    background: var(--bg-card-hover);
    color: var(--text-primary);
  }

  .btn-ghost:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .error-banner {
    padding: 12px 16px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: var(--radius-md);
    color: #ef4444;
    font-size: 13px;
  }

  .email-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }

  .email-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .toolbar-label {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .toolbar-count {
    font-size: 12px;
    color: var(--accent);
    background: var(--accent-subtle);
    padding: 3px 10px;
    border-radius: var(--radius-sm);
  }

  .loading-state,
  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding: 48px 20px;
    color: var(--text-muted);
    font-size: 13px;
  }

  .spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid var(--border-subtle);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .email-list {
    display: flex;
    flex-direction: column;
  }

  .email-row {
    display: grid;
    grid-template-columns: 12px 160px 1fr auto;
    gap: 16px;
    align-items: center;
    padding: 14px 20px;
    border-bottom: 1px solid var(--border-subtle);
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .email-row:last-child {
    border-bottom: none;
  }

  .email-row:hover {
    background: var(--bg-card-hover);
  }

  .email-row.unread {
    background: rgba(124, 58, 237, 0.04);
  }

  .email-row.selected {
    background: rgba(124, 58, 237, 0.08);
  }

  .email-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    opacity: 0;
  }

  .email-dot.visible {
    background: var(--accent);
    opacity: 1;
  }

  .email-from {
    font-size: 13px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .email-row.unread .email-from {
    color: var(--text-primary);
    font-weight: 600;
  }

  .email-subject {
    display: flex;
    gap: 8px;
    overflow: hidden;
  }

  .subject-text {
    font-size: 13px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex-shrink: 0;
    max-width: 50%;
  }

  .email-row.unread .subject-text {
    color: var(--text-primary);
    font-weight: 500;
  }

  .email-snippet {
    font-size: 13px;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .email-time {
    font-size: 11px;
    color: var(--text-muted);
    white-space: nowrap;
  }

  /* Detail Panel */
  .detail-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 100;
    padding: 40px;
  }

  .detail-panel {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 720px;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .detail-subject {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    margin-right: 16px;
  }

  .btn-close {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 24px;
    cursor: pointer;
    padding: 0;
    line-height: 1;
    flex-shrink: 0;
  }

  .btn-close:hover {
    color: var(--text-primary);
  }

  .detail-meta {
    padding: 16px 20px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .detail-from {
    font-size: 13px;
    color: var(--text-primary);
    font-weight: 500;
  }

  .detail-to,
  .detail-date {
    font-size: 12px;
    color: var(--text-muted);
  }

  .detail-body {
    padding: 20px;
    overflow-y: auto;
    flex: 1;
  }

  .body-text {
    font-size: 13px;
    color: var(--text-secondary);
    line-height: 1.6;
    white-space: pre-wrap;
    word-wrap: break-word;
    font-family: inherit;
    margin: 0;
  }

  /* Compose Panel */
  .compose-panel {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 600px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .compose-form {
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .compose-input {
    padding: 10px 14px;
    background: var(--bg-main);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    font-size: 13px;
    outline: none;
  }

  .compose-input:focus {
    border-color: var(--accent);
  }

  .compose-textarea {
    padding: 10px 14px;
    background: var(--bg-main);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    font-size: 13px;
    font-family: inherit;
    resize: vertical;
    outline: none;
  }

  .compose-textarea:focus {
    border-color: var(--accent);
  }

  .compose-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    margin-top: 4px;
  }

  .send-error {
    font-size: 12px;
    color: #ef4444;
  }

  .send-success {
    font-size: 12px;
    color: var(--success);
  }
</style>

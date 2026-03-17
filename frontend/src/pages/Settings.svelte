<script>
  import { onMount, onDestroy } from 'svelte'

  let googleAuth = { authenticated: false, email: '' }
  let userInfo = null
  let loading = false

  async function checkAuth() {
    try {
      googleAuth = await window['go']['main']['App']['CheckGoogleAuth']()
      if (googleAuth.authenticated) {
        userInfo = await window['go']['main']['App']['GetGoogleUserInfo']()
      }
    } catch {
      googleAuth = { authenticated: false, email: '' }
    }
  }

  async function connectGoogle() {
    loading = true
    try {
      const result = await window['go']['main']['App']['StartGoogleLogin']()
      if (result.authenticated) {
        googleAuth = result
        userInfo = await window['go']['main']['App']['GetGoogleUserInfo']()
      }
    } catch {
      // error handled via events
    }
    loading = false
  }

  async function disconnectGoogle() {
    try {
      await window['go']['main']['App']['LogoutGoogle']()
      googleAuth = { authenticated: false, email: '' }
      userInfo = null
    } catch {
      // handled
    }
  }

  onMount(() => {
    checkAuth()
    checkHandshakeStatus()
    const rt = window.runtime
    if (rt) {
      rt.EventsOn('google:authenticated', () => checkAuth())
      rt.EventsOn('google:logged-out', () => {
        googleAuth = { authenticated: false, email: '' }
        userInfo = null
      })
    }
  })

  onDestroy(() => {
    const rt = window.runtime
    if (rt) {
      rt.EventsOff('google:authenticated')
      rt.EventsOff('google:logged-out')
    }
  })

  // Handshake state
  let handshakeStatus = { connected: false, lastHandshake: '' }
  let handshakeState = 'idle' // 'idle' | 'connecting' | 'success' | 'error'
  let handshakeError = ''
  let handshakeResponse = ''

  async function checkHandshakeStatus() {
    try {
      handshakeStatus = await window['go']['main']['App']['CheckHandshake']()
    } catch {
      handshakeStatus = { connected: false, lastHandshake: '' }
    }
  }

  async function forceHandshake() {
    handshakeState = 'connecting'
    handshakeError = ''
    handshakeResponse = ''
    try {
      const result = await window['go']['main']['App']['PerformHandshake']()
      handshakeResponse = result
      handshakeStatus = await window['go']['main']['App']['CheckHandshake']()
      handshakeState = 'success'
    } catch (err) {
      handshakeError = typeof err === 'string' ? err : err?.message || 'Unknown error'
      handshakeState = 'error'
    }
  }

  function formatHandshakeTime(ts) {
    if (!ts) return 'Never'
    try {
      return new Date(ts).toLocaleString('es-MX', { dateStyle: 'medium', timeStyle: 'short' })
    } catch {
      return ts
    }
  }

  const generalItems = [
    { label: 'App Name', value: 'Nyx Command Center', type: 'text' },
    { label: 'Version', value: '0.1.0', type: 'readonly' },
    { label: 'Theme', value: 'Dark', type: 'readonly' },
  ]

  const dbItems = [
    { label: 'MongoDB Atlas URI', value: '', placeholder: 'mongodb+srv://...', type: 'password' },
    { label: 'Database Name', value: '', placeholder: 'nyx-db', type: 'text' },
    { label: 'SQLite Cache Path', value: '~/.nyx/cache.db', type: 'readonly' },
  ]
</script>

<div class="settings-page">
  <div class="settings-header">
    <p class="settings-subtitle">Configure your Nyx Command Center</p>
  </div>

  <!-- General -->
  <div class="settings-section">
    <h2 class="section-title">General</h2>
    <div class="section-card">
      {#each generalItems as item}
        <div class="setting-row">
          <label class="setting-label">{item.label}</label>
          {#if item.type === 'readonly'}
            <span class="setting-value">{item.value}</span>
          {:else}
            <input type="text" class="setting-input" placeholder={item.placeholder || ''} value={item.value} />
          {/if}
        </div>
      {/each}
    </div>
  </div>

  <!-- Database -->
  <div class="settings-section">
    <h2 class="section-title">Database</h2>
    <div class="section-card">
      {#each dbItems as item}
        <div class="setting-row">
          <label class="setting-label">{item.label}</label>
          {#if item.type === 'readonly'}
            <span class="setting-value">{item.value}</span>
          {:else if item.type === 'password'}
            <input type="password" class="setting-input" placeholder={item.placeholder} value={item.value} />
          {:else}
            <input type="text" class="setting-input" placeholder={item.placeholder || ''} value={item.value} />
          {/if}
        </div>
      {/each}
    </div>
  </div>

  <!-- Google Account -->
  <div class="settings-section">
    <h2 class="section-title">Google Account</h2>
    <div class="section-card">
      {#if googleAuth.authenticated && userInfo}
        <div class="setting-row">
          <div class="google-profile">
            {#if userInfo.picture}
              <img class="google-avatar" src={userInfo.picture} alt="avatar" referrerpolicy="no-referrer" />
            {:else}
              <div class="google-avatar-placeholder">{(userInfo.email || '?').charAt(0).toUpperCase()}</div>
            {/if}
            <div class="google-details">
              <span class="google-name">{userInfo.name || 'Google User'}</span>
              <span class="google-email">{userInfo.email}</span>
            </div>
          </div>
          <button class="disconnect-btn" on:click={disconnectGoogle}>Disconnect</button>
        </div>
      {:else}
        <div class="setting-row">
          <label class="setting-label">Google Account</label>
          <div class="connect-row">
            <span class="connect-status">Not Connected</span>
            <button class="connect-btn" on:click={connectGoogle} disabled={loading}>
              {loading ? 'Connecting...' : 'Connect'}
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>

  <!-- Agent Connection -->
  <div class="settings-section">
    <h2 class="section-title">Agent Connection</h2>
    <div class="section-card">
      <div class="setting-row">
        <div class="agent-info">
          <label class="setting-label">OpenClaw Agent</label>
          <span class="setting-sublabel">Handshake sends capabilities to the connected agent so it knows how to use this app.</span>
        </div>
        <div class="agent-status-area">
          {#if handshakeStatus.connected}
            <span class="status-badge connected">Connected</span>
            <span class="last-sync">Last: {formatHandshakeTime(handshakeStatus.lastHandshake)}</span>
          {:else}
            <span class="status-badge disconnected">Not connected</span>
          {/if}
        </div>
      </div>

      <div class="setting-row agent-action-row">
        <div class="agent-info">
          <label class="setting-label">Force Handshake</label>
          <span class="setting-sublabel">Use this after updating the agent capabilities or to reconnect a new OpenClaw instance.</span>
        </div>
        <button
          class="handshake-btn"
          class:connecting={handshakeState === 'connecting'}
          class:success={handshakeState === 'success'}
          on:click={forceHandshake}
          disabled={handshakeState === 'connecting'}
        >
          {#if handshakeState === 'connecting'}
            <span class="btn-spinner"></span> Connecting...
          {:else if handshakeState === 'success'}
            ✓ Connected
          {:else}
            🤝 Run Handshake
          {/if}
        </button>
      </div>

      {#if handshakeState === 'error'}
        <div class="setting-row">
          <span class="error-msg">⚠ {handshakeError}</span>
        </div>
      {/if}

      {#if handshakeState === 'success' && handshakeResponse}
        <div class="setting-row agent-response-row">
          <div class="agent-response">
            <span class="response-label">Agent response:</span>
            <p class="response-text">{handshakeResponse}</p>
          </div>
        </div>
      {/if}
    </div>
  </div>

  <!-- Integrations -->
  <div class="settings-section">
    <h2 class="section-title">Integrations</h2>
    <div class="section-card">
      <div class="setting-row">
        <label class="setting-label">Gmail API</label>
        <div class="connect-row">
          {#if googleAuth.authenticated}
            <span class="status-badge connected">Connected</span>
          {:else}
            <span class="connect-status">Not Connected</span>
            <button class="connect-btn" on:click={connectGoogle} disabled={loading}>Connect</button>
          {/if}
        </div>
      </div>
      <div class="setting-row">
        <label class="setting-label">Google Calendar</label>
        <div class="connect-row">
          {#if googleAuth.authenticated}
            <span class="status-badge connected">Connected</span>
          {:else}
            <span class="connect-status">Not Connected</span>
            <button class="connect-btn" on:click={connectGoogle} disabled={loading}>Connect</button>
          {/if}
        </div>
      </div>
      <div class="setting-row">
        <label class="setting-label">Google Drive</label>
        <div class="connect-row">
          {#if googleAuth.authenticated}
            <span class="status-badge connected">Connected</span>
          {:else}
            <span class="connect-status">Not Connected</span>
            <button class="connect-btn" on:click={connectGoogle} disabled={loading}>Connect</button>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  .settings-page {
    display: flex;
    flex-direction: column;
    gap: 24px;
    max-width: 720px;
  }

  .settings-subtitle {
    font-size: 13px;
    color: var(--text-muted);
  }

  .section-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 12px;
  }

  .section-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }

  .setting-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .setting-row:last-child {
    border-bottom: none;
  }

  .setting-label {
    font-size: 13px;
    color: var(--text-primary);
    font-weight: 500;
  }

  .setting-value {
    font-size: 13px;
    color: var(--text-muted);
  }

  .setting-input {
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 8px 12px;
    color: var(--text-primary);
    font-size: 13px;
    width: 300px;
    outline: none;
    transition: border-color var(--transition-fast);
  }

  .setting-input:focus {
    border-color: var(--accent);
  }

  .setting-input::placeholder {
    color: var(--text-muted);
  }

  .connect-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .connect-status {
    font-size: 12px;
    color: var(--text-muted);
  }

  .connect-btn {
    background: var(--accent);
    color: white;
    border: none;
    border-radius: var(--radius-sm);
    padding: 6px 16px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: background var(--transition-fast);
    font-family: inherit;
  }

  .connect-btn:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .connect-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  /* Google profile */
  .google-profile {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .google-avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    object-fit: cover;
  }

  .google-avatar-placeholder {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: var(--accent);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 16px;
    font-weight: 600;
  }

  .google-details {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .google-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .google-email {
    font-size: 12px;
    color: var(--text-muted);
  }

  .disconnect-btn {
    background: none;
    border: 1px solid var(--error);
    color: var(--error);
    border-radius: var(--radius-sm);
    padding: 6px 16px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: background var(--transition-fast);
    font-family: inherit;
  }

  .disconnect-btn:hover {
    background: rgba(239, 68, 68, 0.1);
  }

  /* Agent connection */
  .agent-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1;
  }

  .setting-sublabel {
    font-size: 11px;
    color: var(--text-muted);
    max-width: 380px;
    line-height: 1.4;
  }

  .agent-status-area {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 4px;
  }

  .last-sync {
    font-size: 11px;
    color: var(--text-muted);
  }

  .agent-action-row {
    align-items: flex-start;
    padding-top: 18px;
    padding-bottom: 18px;
  }

  .handshake-btn {
    background: var(--accent-subtle);
    border: 1px solid var(--accent);
    border-radius: var(--radius-sm);
    color: var(--accent);
    font-size: 12px;
    font-weight: 600;
    padding: 8px 18px;
    cursor: pointer;
    font-family: inherit;
    display: flex;
    align-items: center;
    gap: 6px;
    white-space: nowrap;
    transition: background var(--transition-fast), color var(--transition-fast);
    flex-shrink: 0;
  }

  .handshake-btn:hover:not(:disabled) {
    background: var(--accent);
    color: white;
  }

  .handshake-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .handshake-btn.success {
    background: rgba(34, 197, 94, 0.1);
    border-color: rgba(34, 197, 94, 0.4);
    color: var(--success);
  }

  .btn-spinner {
    width: 11px;
    height: 11px;
    border: 2px solid var(--accent);
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    display: inline-block;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .error-msg {
    font-size: 12px;
    color: var(--error);
  }

  .agent-response-row {
    display: block;
    padding: 16px 20px;
  }

  .agent-response {
    width: 100%;
  }

  .response-label {
    font-size: 11px;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    display: block;
    margin-bottom: 8px;
  }

  .response-text {
    font-size: 12px;
    color: var(--text-secondary);
    background: var(--bg-input);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: 12px;
    margin: 0;
    line-height: 1.6;
    white-space: pre-wrap;
    max-height: 200px;
    overflow-y: auto;
  }

  /* Status badges */
  .status-badge {
    font-size: 12px;
    font-weight: 500;
    padding: 4px 10px;
    border-radius: var(--radius-sm);
  }

  .status-badge.connected {
    background: rgba(34, 197, 94, 0.1);
    color: var(--success);
    border: 1px solid rgba(34, 197, 94, 0.3);
  }

  .status-badge.disconnected {
    background: rgba(156, 163, 175, 0.1);
    color: var(--text-muted);
    border: 1px solid rgba(156, 163, 175, 0.2);
  }
</style>

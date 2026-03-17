<script>
  import { onMount, onDestroy } from 'svelte'

  export let title = 'Dashboard'
  export let userEmail = ''

  let statusOnline = true
  let healthColor = '#888'
  let userInfo = null
  let showDropdown = false

  // Handshake state: 'idle' | 'connecting' | 'connected' | 'minimized'
  let handshakeState = 'idle'

  // Sync state
  let syncActive = false
  let lastSyncAgo = ''
  let syncTimer = null

  async function checkHandshake() {
    try {
      const status = await window['go']['main']['App']['CheckHandshake']()
      if (status?.connected) {
        handshakeState = 'minimized'
      }
    } catch {
      // not connected
    }
  }

  let handshakeError = ''

  async function resetHandshake() {
    try {
      await window['go']['main']['App']['ResetHandshake']()
      handshakeState = 'idle'
    } catch (err) {
      console.error('[Handshake] Reset error:', err)
    }
  }

  async function performHandshake() {
    handshakeState = 'connecting'
    handshakeError = ''
    try {
      console.log('[Handshake] Starting...')
      console.log('[Handshake] OPENCLAW_URL and token should be set in .env')
      const result = await window['go']['main']['App']['PerformHandshake']()
      console.log('[Handshake] Success:', result)
      handshakeState = 'connected'
      setTimeout(() => { handshakeState = 'minimized' }, 3000)
    } catch (err) {
      console.error('[Handshake] Error:', err)
      handshakeError = typeof err === 'string' ? err : err?.message || 'Unknown error'
      handshakeState = 'error'
      setTimeout(() => { handshakeState = 'idle' }, 5000)
    }
  }

  async function loadUserInfo() {
    if (!userEmail) return
    try {
      userInfo = await window['go']['main']['App']['GetGoogleUserInfo']()
    } catch {
      userInfo = null
    }
  }

  async function handleSignOut() {
    showDropdown = false
    try {
      await window['go']['main']['App']['LogoutGoogle']()
    } catch {
      // event will handle state
    }
  }

  function toggleDropdown() {
    showDropdown = !showDropdown
  }

  function handleClickOutside(e) {
    if (showDropdown && !e.target.closest('.profile-area')) {
      showDropdown = false
    }
  }

  function updateSyncState() {
    const wails = window['go']?.['main']?.['App']
    if (!wails?.GetSyncState) return
    wails.GetSyncState().then(state => {
      if (state?.lastSyncTime) {
        const diff = Date.now() - new Date(state.lastSyncTime).getTime()
        const secs = Math.floor(diff / 1000)
        lastSyncAgo = secs < 60 ? secs + 's ago' : Math.floor(secs / 60) + 'm ago'
      } else {
        lastSyncAgo = 'never'
      }
    }).catch(() => {})
  }

  onMount(() => {
    const runtime = window.runtime
    if (runtime) {
      runtime.EventsOn('health:report', (report) => {
        if (!report) return
        switch (report.overall) {
          case 'ok':
            healthColor = '#22c55e'
            break
          case 'degraded':
            healthColor = '#eab308'
            break
          case 'error':
            healthColor = '#ef4444'
            break
          default:
            healthColor = '#888'
        }
      })
      runtime.EventsOn('google:authenticated', () => {
        loadUserInfo()
      })
      runtime.EventsOn('hivemind:new-activity', () => {
        syncActive = true
        setTimeout(() => { syncActive = false }, 1500)
        updateSyncState()
      })
    }
    if (userEmail) loadUserInfo()
    checkHandshake()
    updateSyncState()
    syncTimer = setInterval(updateSyncState, 10000)
    document.addEventListener('click', handleClickOutside)
  })

  onDestroy(() => {
    const runtime = window.runtime
    if (runtime) {
      runtime.EventsOff('health:report')
      runtime.EventsOff('google:authenticated')
      runtime.EventsOff('hivemind:new-activity')
    }
    if (syncTimer) clearInterval(syncTimer)
    document.removeEventListener('click', handleClickOutside)
  })

  $: if (userEmail) loadUserInfo()
</script>

<header class="header">
  <div class="header-left">
    <h1 class="page-title">{title}</h1>
  </div>
  <div class="header-right">
    <div class="header-indicator" title="System Health: verifica OpenClaw Gateway, MongoDB y Google OAuth. Verde = todo bien, amarillo = degradado, rojo = error.">
      <span class="indicator-label">SYS</span>
      <div class="health-dot" style="background: {healthColor}; box-shadow: 0 0 6px {healthColor}40;"></div>
    </div>

    <div class="sync-indicator" class:syncing={syncActive} title="Hive Mind Sync — sincronización entre agentes activos. {syncActive ? 'Sincronizando ahora...' : 'Última sync: ' + lastSyncAgo}">
      <span class="sync-icon">{syncActive ? '↻' : '⬡'}</span>
    </div>

    {#if handshakeState === 'idle'}
      <button class="handshake-btn pulse" on:click={performHandshake} title="Conectar agente — envía las capacidades de Nyx al agente OpenClaw para que sepa cómo usar esta app.">
        <span class="handshake-icon">🤝</span>
        <span class="handshake-text">Connect Agent</span>
      </button>
    {:else if handshakeState === 'connecting'}
      <div class="handshake-btn connecting" title="Estableciendo conexión con el agente...">
        <span class="handshake-spinner"></span>
        <span class="handshake-text">Connecting...</span>
      </div>
    {:else if handshakeState === 'connected'}
      <div class="handshake-badge connected" title="Agente conectado correctamente.">
        <span class="handshake-dot"></span>
        <span class="handshake-text">Agent Connected</span>
      </div>
    {:else if handshakeState === 'error'}
      <div class="handshake-badge error" title="Error al conectar: {handshakeError}">
        <span class="handshake-error-icon">⚠</span>
        <span class="handshake-text">{handshakeError.substring(0, 40)}</span>
      </div>
    {:else if handshakeState === 'minimized'}
      <div class="handshake-connected-area" title="Agente OpenClaw conectado. Hover para reconectar.">
        <span class="indicator-label agent-label">AI</span>
        <div class="handshake-dot-only"></div>
        <button class="handshake-reset-btn" on:click={resetHandshake} title="Forzar reconexión del agente">↺</button>
      </div>
    {/if}

    <div class="status-indicator" class:online={statusOnline} class:offline={!statusOnline} title="Estado de red — {statusOnline ? 'conectado a internet' : 'sin conexión'}">
      <span class="status-dot"></span>
      <span class="status-text">{statusOnline ? 'Online' : 'Offline'}</span>
    </div>
    <div class="header-divider"></div>

    {#if userEmail}
      <div class="profile-area">
        <button class="profile-btn" on:click={toggleDropdown}>
          {#if userInfo?.picture}
            <img class="profile-avatar" src={userInfo.picture} alt="avatar" referrerpolicy="no-referrer" />
          {:else}
            <div class="profile-avatar-placeholder">
              {userEmail.charAt(0).toUpperCase()}
            </div>
          {/if}
          <span class="profile-email">{userInfo?.name || userEmail}</span>
          <span class="profile-chevron">{showDropdown ? '\u25B4' : '\u25BE'}</span>
        </button>

        {#if showDropdown}
          <div class="profile-dropdown">
            <div class="dropdown-header">
              <span class="dropdown-email">{userEmail}</span>
            </div>
            <div class="dropdown-divider"></div>
            <button class="dropdown-item" on:click={handleSignOut}>
              <span class="dropdown-icon">&#x2190;</span>
              Sign out
            </button>
          </div>
        {/if}
      </div>
    {:else}
      <div class="app-badge">
        <span class="badge-text">Nyx v0.1.0</span>
      </div>
    {/if}
  </div>
</header>

<style>
  .header {
    height: var(--header-height);
    background: var(--bg-sidebar);
    border-bottom: 1px solid var(--border-subtle);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 24px;
    flex-shrink: 0;
  }

  .header-left {
    display: flex;
    align-items: center;
  }

  .page-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--text-primary);
    letter-spacing: 0.3px;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .header-indicator {
    display: flex;
    align-items: center;
    gap: 5px;
    cursor: default;
  }

  .indicator-label {
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.5px;
    color: var(--text-muted);
    text-transform: uppercase;
    line-height: 1;
  }

  .agent-label {
    color: #22c55e;
    opacity: 0.8;
  }

  .health-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    transition: background 0.3s, box-shadow 0.3s;
  }

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--text-muted);
  }

  .status-indicator.online .status-dot {
    background: var(--success);
    box-shadow: 0 0 6px rgba(34, 197, 94, 0.4);
  }

  .status-indicator.offline .status-dot {
    background: var(--error);
  }

  .status-text {
    font-size: 12px;
    color: var(--text-secondary);
  }

  .header-divider {
    width: 1px;
    height: 20px;
    background: var(--border);
  }

  .app-badge {
    background: var(--accent-subtle);
    padding: 4px 10px;
    border-radius: var(--radius-sm);
  }

  .badge-text {
    font-size: 11px;
    color: var(--accent);
    font-weight: 600;
    letter-spacing: 0.5px;
  }

  /* Profile area */
  .profile-area {
    position: relative;
  }

  .profile-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    background: none;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: 4px 10px 4px 4px;
    cursor: pointer;
    transition: background var(--transition-fast);
    color: var(--text-primary);
    font-family: inherit;
  }

  .profile-btn:hover {
    background: var(--bg-card);
  }

  .profile-avatar {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    object-fit: cover;
  }

  .profile-avatar-placeholder {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: var(--accent);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 13px;
    font-weight: 600;
  }

  .profile-email {
    font-size: 12px;
    color: var(--text-secondary);
    max-width: 140px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .profile-chevron {
    font-size: 10px;
    color: var(--text-muted);
  }

  .profile-dropdown {
    position: absolute;
    top: calc(100% + 8px);
    right: 0;
    min-width: 200px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    z-index: 100;
    overflow: hidden;
  }

  .dropdown-header {
    padding: 12px 16px;
  }

  .dropdown-email {
    font-size: 12px;
    color: var(--text-muted);
    word-break: break-all;
  }

  .dropdown-divider {
    height: 1px;
    background: var(--border-subtle);
  }

  .dropdown-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 10px 16px;
    background: none;
    border: none;
    color: var(--text-primary);
    font-size: 13px;
    cursor: pointer;
    transition: background var(--transition-fast);
    font-family: inherit;
    text-align: left;
  }

  .dropdown-item:hover {
    background: var(--bg-card-hover);
  }

  .dropdown-icon {
    font-size: 14px;
    color: var(--text-muted);
  }

  /* Handshake */
  .handshake-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--accent-subtle);
    border: 1px solid var(--accent);
    border-radius: var(--radius-sm);
    padding: 4px 10px;
    cursor: pointer;
    font-family: inherit;
    font-size: 11px;
    color: var(--accent);
    font-weight: 600;
    transition: background var(--transition-fast), opacity var(--transition-fast);
  }

  .handshake-btn:hover {
    background: var(--accent);
    color: #fff;
  }

  .handshake-btn.pulse {
    animation: handshake-pulse 2s ease-in-out infinite;
  }

  @keyframes handshake-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.6; }
  }

  .handshake-icon {
    font-size: 13px;
    line-height: 1;
  }

  .handshake-text {
    letter-spacing: 0.3px;
  }

  .handshake-btn.connecting {
    cursor: default;
    opacity: 0.8;
  }

  .handshake-spinner {
    width: 12px;
    height: 12px;
    border: 2px solid var(--accent);
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .handshake-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    border-radius: var(--radius-sm);
    font-size: 11px;
    font-weight: 600;
    color: #22c55e;
    background: rgba(34, 197, 94, 0.1);
    border: 1px solid rgba(34, 197, 94, 0.3);
    animation: handshake-fade-in 0.3s ease;
  }

  @keyframes handshake-fade-in {
    from { opacity: 0; transform: scale(0.95); }
    to { opacity: 1; transform: scale(1); }
  }

  .handshake-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #22c55e;
    box-shadow: 0 0 6px rgba(34, 197, 94, 0.4);
  }

  .handshake-badge.error {
    color: #ef4444;
    background: rgba(239, 68, 68, 0.1);
    border-color: rgba(239, 68, 68, 0.3);
    font-size: 10px;
    max-width: 300px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .handshake-error-icon {
    font-size: 12px;
  }

  .handshake-connected-area {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .handshake-dot-only {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #22c55e;
    box-shadow: 0 0 6px rgba(34, 197, 94, 0.4);
    animation: handshake-fade-in 0.3s ease;
  }

  .handshake-reset-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 12px;
    cursor: pointer;
    padding: 0 2px;
    line-height: 1;
    opacity: 0;
    transition: opacity 0.2s;
  }

  .handshake-connected-area:hover .handshake-reset-btn {
    opacity: 1;
  }

  .handshake-reset-btn:hover {
    color: var(--text-secondary);
  }

  /* Sync Indicator */
  .sync-indicator {
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    transition: all 0.3s;
    cursor: default;
  }

  .sync-icon {
    font-size: 12px;
    color: var(--text-muted);
    transition: color 0.3s;
  }

  .sync-indicator.syncing .sync-icon {
    color: var(--accent);
    animation: spin 0.8s linear infinite;
  }
</style>

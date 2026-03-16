<script>
  import { onMount, onDestroy } from 'svelte'

  export let title = 'Dashboard'
  export let userEmail = ''

  let statusOnline = true
  let healthColor = '#888'
  let userInfo = null
  let showDropdown = false

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
    }
    if (userEmail) loadUserInfo()
    document.addEventListener('click', handleClickOutside)
  })

  onDestroy(() => {
    const runtime = window.runtime
    if (runtime) {
      runtime.EventsOff('health:report')
      runtime.EventsOff('google:authenticated')
    }
    document.removeEventListener('click', handleClickOutside)
  })

  $: if (userEmail) loadUserInfo()
</script>

<header class="header">
  <div class="header-left">
    <h1 class="page-title">{title}</h1>
  </div>
  <div class="header-right">
    <div class="health-dot" style="background: {healthColor}; box-shadow: 0 0 6px {healthColor}40;" title="System Health"></div>
    <div class="status-indicator" class:online={statusOnline} class:offline={!statusOnline}>
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

  .health-dot {
    width: 10px;
    height: 10px;
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
</style>

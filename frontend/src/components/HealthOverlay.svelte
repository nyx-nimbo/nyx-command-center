<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()
  const runtime = window.runtime

  let visible = true
  let fading = false
  let report = null
  let services = [
    { name: 'OpenClaw Gateway', status: 'checking', message: 'Checking...' },
    { name: 'MongoDB', status: 'checking', message: 'Checking...' },
    { name: 'Google OAuth', status: 'checking', message: 'Checking...' },
  ]

  function statusIcon(status) {
    switch (status) {
      case 'checking': return '\u23F3'
      case 'ok': return '\u2705'
      case 'repairing': return '\uD83D\uDD27'
      case 'error': return '\u274C'
      default: return '\u23F3'
    }
  }

  // Timeout: auto-dismiss after 15s if checks never complete (stuck overlay fix)
  let healthTimeout

  onMount(() => {
    healthTimeout = setTimeout(() => {
      if (visible && !allDone) {
        services = services.map(s =>
          s.status === 'checking' ? { ...s, status: 'error', message: 'Health check timed out' } : s
        )
      }
    }, 15000)

    if (runtime) {
      runtime.EventsOn('health:report', (r) => {
        report = r
        if (r && r.services) {
          services = r.services
        }
        if (r && r.overall === 'ok') {
          clearTimeout(healthTimeout)
          setTimeout(() => {
            fading = true
            setTimeout(() => {
              visible = false
              dispatch('done', report)
            }, 500)
          }, 1500)
        }
      })

      runtime.EventsOn('health:repairing', (service) => {
        services = services.map(s =>
          s.name === service ? { ...s, status: 'repairing', message: 'Auto-repairing...' } : s
        )
      })

      runtime.EventsOn('health:repaired', (service) => {
        services = services.map(s =>
          s.name === service ? { ...s, status: 'ok', message: 'Repaired' } : s
        )
      })
    } else {
      // No Wails runtime (browser dev mode) — skip health check
      setTimeout(() => {
        fading = true
        setTimeout(() => {
          visible = false
          dispatch('done', null)
        }, 500)
      }, 500)
    }
  })

  onDestroy(() => {
    clearTimeout(healthTimeout)
    if (runtime) {
      runtime.EventsOff('health:report')
      runtime.EventsOff('health:repairing')
      runtime.EventsOff('health:repaired')
    }
  })

  async function retry() {
    services = services.map(s => ({ ...s, status: 'checking', message: 'Checking...' }))
    const wails = window['go']?.['main']?.['App']
    if (wails) {
      await wails.CheckHealth()
    }
  }

  function dismiss() {
    fading = true
    setTimeout(() => {
      visible = false
      dispatch('done', report)
    }, 500)
  }

  $: hasErrors = services.some(s => s.status === 'error')
  $: allDone = services.every(s => s.status === 'ok' || s.status === 'error')
</script>

{#if visible}
  <div class="overlay" class:fading>
    <div class="health-card">
      <div class="health-logo">
        <span class="logo-hex">&#x2B21;</span>
        <span class="logo-text">NYX</span>
      </div>
      <h2>System Health Check</h2>

      <div class="checklist">
        {#each services as svc}
          <div class="check-item" class:ok={svc.status === 'ok'} class:error={svc.status === 'error'} class:repairing={svc.status === 'repairing'}>
            <span class="check-icon" class:spin={svc.status === 'checking' || svc.status === 'repairing'}>{statusIcon(svc.status)}</span>
            <div class="check-info">
              <span class="check-name">{svc.name}</span>
              <span class="check-msg">{svc.message}</span>
            </div>
          </div>
        {/each}
      </div>

      {#if hasErrors && allDone}
        <div class="health-actions">
          <button class="retry-btn" on:click={retry}>Retry</button>
          <button class="dismiss-btn" on:click={dismiss}>Continue Anyway</button>
        </div>
      {/if}

      {#if !allDone}
        <div class="checking-label">Running checks...</div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.92);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
    transition: opacity 0.5s ease;
  }

  .overlay.fading {
    opacity: 0;
    pointer-events: none;
  }

  .health-card {
    background: #141422;
    border: 1px solid #2a2a3e;
    border-radius: 16px;
    padding: 40px;
    width: 420px;
    max-width: 90vw;
    text-align: center;
  }

  .health-logo {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    margin-bottom: 20px;
  }

  .logo-hex {
    font-size: 32px;
    color: #7c3aed;
  }

  .logo-text {
    font-size: 20px;
    font-weight: 700;
    letter-spacing: 4px;
    color: #e0e0e0;
  }

  h2 {
    font-size: 14px;
    font-weight: 500;
    color: #888;
    margin: 0 0 24px 0;
    letter-spacing: 0.5px;
  }

  .checklist {
    display: flex;
    flex-direction: column;
    gap: 12px;
    text-align: left;
  }

  .check-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: #0f0f1a;
    border-radius: 10px;
    border: 1px solid #1a1a2e;
    transition: border-color 0.3s;
  }

  .check-item.ok {
    border-color: rgba(34, 197, 94, 0.3);
  }

  .check-item.error {
    border-color: rgba(239, 68, 68, 0.3);
  }

  .check-item.repairing {
    border-color: rgba(234, 179, 8, 0.3);
  }

  .check-icon {
    font-size: 18px;
    width: 28px;
    text-align: center;
    flex-shrink: 0;
  }

  .check-icon.spin {
    animation: pulse 1.2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  .check-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .check-name {
    font-size: 13px;
    font-weight: 500;
    color: #e0e0e0;
  }

  .check-msg {
    font-size: 11px;
    color: #666;
  }

  .health-actions {
    display: flex;
    gap: 8px;
    justify-content: center;
    margin-top: 24px;
  }

  .retry-btn, .dismiss-btn {
    padding: 8px 20px;
    border-radius: 8px;
    font-size: 13px;
    cursor: pointer;
    border: none;
    transition: all 0.15s;
  }

  .retry-btn {
    background: #7c3aed;
    color: white;
  }

  .retry-btn:hover {
    opacity: 0.85;
  }

  .dismiss-btn {
    background: transparent;
    border: 1px solid #2a2a3e;
    color: #888;
  }

  .dismiss-btn:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  .checking-label {
    margin-top: 20px;
    font-size: 12px;
    color: #555;
    animation: pulse 1.5s ease-in-out infinite;
  }
</style>

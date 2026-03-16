<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()

  let loading = false
  let error = ''

  async function handleGoogleLogin() {
    loading = true
    error = ''
    try {
      const result = await window['go']['main']['App']['StartGoogleLogin']()
      if (result.authenticated) {
        dispatch('authenticated', result)
      } else {
        error = 'Authentication was not completed. Please try again.'
      }
    } catch (err) {
      error = err.message || 'Login failed. Please try again.'
    } finally {
      loading = false
    }
  }

  onMount(() => {
    const rt = window.runtime
    if (rt) {
      rt.EventsOn('google:error', (msg) => {
        error = msg
        loading = false
      })
    }
  })

  onDestroy(() => {
    const rt = window.runtime
    if (rt) {
      rt.EventsOff('google:error')
    }
  })
</script>

<div class="login-overlay">
  <div class="login-container">
    <div class="login-logo">
      <span class="logo-icon">&#x2B21;</span>
    </div>
    <h1 class="login-title">Nyx Command Center</h1>
    <p class="login-subtitle">Sign in to continue</p>

    {#if error}
      <div class="login-error">
        <span class="error-icon">!</span>
        <span>{error}</span>
      </div>
    {/if}

    <button
      class="google-btn"
      on:click={handleGoogleLogin}
      disabled={loading}
    >
      {#if loading}
        <span class="spinner"></span>
        <span>Signing in...</span>
      {:else}
        <svg class="google-icon" viewBox="0 0 24 24" width="18" height="18">
          <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
          <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
          <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
          <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
        </svg>
        <span>Sign in with Google</span>
      {/if}
    </button>

    <p class="login-footer">Secure authentication via Google OAuth 2.0</p>
  </div>
</div>

<style>
  .login-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: var(--bg-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
  }

  .login-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    max-width: 380px;
    width: 100%;
    padding: 48px 32px;
  }

  .login-logo {
    width: 72px;
    height: 72px;
    background: var(--accent-subtle);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 8px;
  }

  .logo-icon {
    font-size: 36px;
    color: var(--accent);
    line-height: 1;
  }

  .login-title {
    font-size: 24px;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.3px;
  }

  .login-subtitle {
    font-size: 14px;
    color: var(--text-muted);
    margin-bottom: 16px;
  }

  .login-error {
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: var(--radius-md);
    padding: 10px 16px;
    width: 100%;
    font-size: 13px;
    color: var(--error);
  }

  .error-icon {
    width: 20px;
    height: 20px;
    background: var(--error);
    color: white;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 700;
    flex-shrink: 0;
  }

  .google-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    width: 100%;
    padding: 12px 24px;
    background: #fff;
    color: #3c4043;
    border: 1px solid #dadce0;
    border-radius: var(--radius-md);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: background var(--transition-fast), box-shadow var(--transition-fast);
    font-family: inherit;
  }

  .google-btn:hover:not(:disabled) {
    background: #f8f9fa;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  }

  .google-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .google-icon {
    flex-shrink: 0;
  }

  .spinner {
    width: 18px;
    height: 18px;
    border: 2px solid #dadce0;
    border-top-color: #4285F4;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .login-footer {
    font-size: 11px;
    color: var(--text-muted);
    margin-top: 8px;
  }
</style>

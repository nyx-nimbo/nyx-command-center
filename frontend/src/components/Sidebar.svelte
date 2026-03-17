<script>
  import { link } from 'svelte-spa-router'
  import { location } from 'svelte-spa-router'

  export let collapsed = false

  const navItems = [
    { path: '/chat', label: 'Chat', icon: '⬡' },
    { path: '/', label: 'Dashboard', icon: '◈' },
    { path: '/projects', label: 'Projects', icon: '◧' },
    { path: '/clients', label: 'Clients', icon: '▣' },
    { path: '/ideas', label: 'Ideas', icon: '◆' },
    { path: '/email', label: 'Email', icon: '✉' },
    { path: '/calendar', label: 'Calendar', icon: '▦' },
    { path: '/settings', label: 'Settings', icon: '⚙' },
  ]

  function toggleCollapse() {
    collapsed = !collapsed
  }
</script>

<aside class="sidebar" class:collapsed>
  <div class="sidebar-header">
    {#if !collapsed}
      <div class="logo">
        <span class="logo-icon">⬡</span>
        <span class="logo-text">NYX</span>
      </div>
    {:else}
      <div class="logo">
        <span class="logo-icon">⬡</span>
      </div>
    {/if}
  </div>

  <nav class="nav">
    {#each navItems as item}
      <a
        href={item.path}
        use:link
        class="nav-item"
        class:active={$location === item.path || (item.path === '/projects' && $location.startsWith('/project/'))}
        title={collapsed ? item.label : ''}
      >
        <span class="nav-icon">{item.icon}</span>
        {#if !collapsed}
          <span class="nav-label">{item.label}</span>
        {/if}
      </a>
    {/each}
  </nav>

  <button class="collapse-btn" on:click={toggleCollapse}>
    {#if collapsed}
      <span class="collapse-icon">▸</span>
    {:else}
      <span class="collapse-icon">◂</span>
      <span class="collapse-label">Collapse</span>
    {/if}
  </button>
</aside>

<style>
  .sidebar {
    width: var(--sidebar-width);
    height: 100vh;
    background: var(--bg-sidebar);
    border-right: 1px solid var(--border-subtle);
    display: flex;
    flex-direction: column;
    transition: width var(--transition-normal);
    overflow: hidden;
    flex-shrink: 0;
  }

  .sidebar.collapsed {
    width: var(--sidebar-collapsed-width);
  }

  .sidebar-header {
    height: var(--header-height);
    display: flex;
    align-items: center;
    padding: 0 16px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .logo-icon {
    font-size: 22px;
    color: var(--accent);
  }

  .logo-text {
    font-size: 16px;
    font-weight: 700;
    letter-spacing: 3px;
    color: var(--text-primary);
  }

  .nav {
    flex: 1;
    padding: 12px 8px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    text-decoration: none;
    transition: all var(--transition-fast);
    white-space: nowrap;
  }

  .nav-item:hover {
    background: var(--accent-subtle);
    color: var(--text-primary);
  }

  .nav-item.active {
    background: var(--accent-subtle);
    color: var(--accent);
  }

  .nav-icon {
    font-size: 16px;
    width: 24px;
    text-align: center;
    flex-shrink: 0;
  }

  .nav-label {
    font-size: 13px;
    font-weight: 500;
  }

  .collapse-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    border: none;
    border-top: 1px solid var(--border-subtle);
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    transition: color var(--transition-fast);
    font-size: 13px;
    white-space: nowrap;
  }

  .collapse-btn:hover {
    color: var(--text-secondary);
  }

  .collapse-icon {
    font-size: 14px;
    width: 24px;
    text-align: center;
  }
</style>

<script>
  import Sidebar from './Sidebar.svelte'
  import Header from './Header.svelte'
  import { location } from 'svelte-spa-router'

  export let userEmail = ''

  let collapsed = false

  const pageTitles = {
    '/': 'Dashboard',
    '/chat': 'Chat',
    '/ideas': 'Ideas',
    '/email': 'Email',
    '/calendar': 'Calendar',
    '/settings': 'Settings',
  }

  $: currentTitle = pageTitles[$location] || 'Dashboard'
</script>

<div class="layout">
  <Sidebar bind:collapsed />
  <div class="main-area">
    <Header title={currentTitle} {userEmail} />
    <main class="content">
      <slot />
    </main>
  </div>
</div>

<style>
  .layout {
    display: flex;
    height: 100vh;
    overflow: hidden;
  }

  .main-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .content {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
  }
</style>

<script>
  import { onMount, onDestroy } from 'svelte'
  import Router from 'svelte-spa-router'
  import { wrap } from 'svelte-spa-router/wrap'
  import Layout from './components/Layout.svelte'
  import HealthOverlay from './components/HealthOverlay.svelte'
  import LoginScreen from './components/LoginScreen.svelte'
  import Dashboard from './pages/Dashboard.svelte'
  import Ideas from './pages/Ideas.svelte'
  import Email from './pages/Email.svelte'
  import Calendar from './pages/Calendar.svelte'
  import Settings from './pages/Settings.svelte'
  import Chat from './pages/Chat.svelte'
  import Clients from './pages/Clients.svelte'
  import Projects from './pages/Projects.svelte'
  import Project from './pages/Project.svelte'

  let showHealth = true
  let authenticated = false
  let authChecked = false
  let userEmail = ''

  const routes = {
    '/': Dashboard,
    '/chat': Chat,
    '/clients': Clients,
    '/projects': Projects,
    '/project/:id': wrap({ component: Project }),
    '/ideas': Ideas,
    '/email': Email,
    '/calendar': Calendar,
    '/settings': Settings,
  }

  function onHealthDone() {
    showHealth = false
  }

  function onAuthenticated(e) {
    authenticated = true
    userEmail = e.detail?.email || ''
  }

  onMount(async () => {
    try {
      const status = await window['go']['main']['App']['CheckGoogleAuth']()
      authenticated = status.authenticated
      userEmail = status.email || ''
    } catch {
      authenticated = false
    }
    authChecked = true

    const rt = window.runtime
    if (rt) {
      rt.EventsOn('google:authenticated', (email) => {
        authenticated = true
        userEmail = email || ''
      })
      rt.EventsOn('google:logged-out', () => {
        authenticated = false
        userEmail = ''
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
</script>

{#if authChecked && !authenticated}
  <LoginScreen on:authenticated={onAuthenticated} />
{:else}
  {#if showHealth}
    <HealthOverlay on:done={onHealthDone} />
  {/if}

  <Layout {userEmail}>
    <Router {routes} />
  </Layout>
{/if}

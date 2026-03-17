<script>
  import { onMount } from 'svelte'

  export let projectId = ''

  const wails = window['go']?.['main']?.['App']

  let assignedAgents = []
  let allAgents = []
  let assignments = []
  let loading = true

  // Add member modal
  let addModal = false
  let selectedAgentId = ''
  let selectedRole = 'developer'

  onMount(async () => {
    await loadData()
  })

  async function loadData() {
    loading = true
    try {
      const [agents, assigned, roles] = await Promise.all([
        wails.GetAgents(),
        wails.GetProjectAssignments(projectId),
        wails.GetProjectAssignmentsWithRoles(projectId),
      ])
      allAgents = agents || []
      assignedAgents = assigned || []
      assignments = roles || []
    } catch (e) {
      console.error('Failed to load team data:', e)
    }
    loading = false
  }

  $: availableAgents = allAgents.filter(a => !assignedAgents.find(aa => aa.agentId === a.agentId))

  function getRole(agentId) {
    const a = assignments.find(pa => pa.agentId === agentId)
    return a ? a.role : ''
  }

  function roleColor(role) {
    return { lead: '#f59e0b', reviewer: '#8b5cf6', developer: '#3b82f6' }[role] || '#71717a'
  }

  function statusColor(status) {
    return { online: '#22c55e', busy: '#f59e0b', offline: '#71717a' }[status] || '#71717a'
  }

  function typeIcon(type) {
    return type === 'agent' ? '⚙' : '○'
  }

  function timeSince(dateStr) {
    if (!dateStr) return 'never'
    const diff = Date.now() - new Date(dateStr).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 60) return mins + 'm ago'
    const hrs = Math.floor(mins / 60)
    if (hrs < 24) return hrs + 'h ago'
    const days = Math.floor(hrs / 24)
    return days + 'd ago'
  }

  async function addMember() {
    if (!selectedAgentId || !selectedRole) return
    try {
      await wails.AssignAgentToProject(projectId, selectedAgentId, selectedRole)
      addModal = false
      selectedAgentId = ''
      selectedRole = 'developer'
      await loadData()
    } catch (e) {
      console.error('Failed to assign agent:', e)
    }
  }

  async function removeMember(agentId) {
    try {
      await wails.UnassignAgentFromProject(projectId, agentId)
      await loadData()
    } catch (e) {
      console.error('Failed to unassign agent:', e)
    }
  }
</script>

<div class="team-panel">
  {#if loading}
    <div class="tp-loading">Loading team...</div>
  {:else}
    <div class="tp-toolbar">
      <div class="tp-count">{assignedAgents.length} member{assignedAgents.length !== 1 ? 's' : ''}</div>
      <button class="tp-btn-primary" on:click={() => { addModal = true; selectedAgentId = ''; selectedRole = 'developer' }}>
        + Add Member
      </button>
    </div>

    {#if assignedAgents.length === 0}
      <div class="tp-empty">
        <p>No team members assigned yet.</p>
        <p class="tp-empty-sub">Add agents or users to this project team.</p>
      </div>
    {:else}
      <div class="tp-list">
        {#each assignedAgents as agent}
          {@const role = getRole(agent.agentId)}
          <div class="tp-row">
            <div class="tp-avatar">
              <span class="tp-type-icon">{typeIcon(agent.type)}</span>
              <span class="tp-status-dot" style="background: {statusColor(agent.status)}"></span>
            </div>
            <div class="tp-info">
              <div class="tp-name">{agent.name}</div>
              <div class="tp-meta">
                <span class="tp-agent-id">{agent.agentId}</span>
                <span class="tp-sep">·</span>
                <span class="tp-last-seen">Last seen {timeSince(agent.lastSeen)}</span>
              </div>
            </div>
            <div class="tp-badges">
              <span class="tp-badge tp-type-badge">{agent.type}</span>
              {#if role}
                <span class="tp-badge tp-role-badge" style="border-color: {roleColor(role)}; color: {roleColor(role)}">{role}</span>
              {/if}
              <span class="tp-badge tp-status-badge" style="border-color: {statusColor(agent.status)}; color: {statusColor(agent.status)}">{agent.status}</span>
            </div>
            <button class="tp-remove" on:click={() => removeMember(agent.agentId)} title="Remove from project">x</button>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<!-- Add Member Modal -->
{#if addModal}
  <div class="tp-overlay" on:click|self={() => addModal = false}>
    <div class="tp-modal">
      <h3>Add Team Member</h3>
      {#if availableAgents.length === 0}
        <p class="tp-empty-modal">All registered agents are already assigned to this project.</p>
      {:else}
        <div class="tp-form-group">
          <label>Agent</label>
          <select bind:value={selectedAgentId}>
            <option value="">Select an agent...</option>
            {#each availableAgents as agent}
              <option value={agent.agentId}>
                {agent.name} ({agent.agentId}) — {agent.type}
              </option>
            {/each}
          </select>
        </div>
        <div class="tp-form-group">
          <label>Role</label>
          <select bind:value={selectedRole}>
            <option value="developer">Developer</option>
            <option value="reviewer">Reviewer</option>
            <option value="lead">Lead</option>
          </select>
        </div>
      {/if}
      <div class="tp-modal-actions">
        <button class="tp-btn-secondary" on:click={() => addModal = false}>Cancel</button>
        {#if availableAgents.length > 0}
          <button class="tp-btn-primary" on:click={addMember} disabled={!selectedAgentId}>Add</button>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .team-panel {
    padding: 0;
  }

  .tp-loading {
    text-align: center;
    color: var(--text-secondary);
    padding: 48px 0;
  }

  .tp-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 0;
    border-bottom: 1px solid var(--border-subtle);
    margin-bottom: 12px;
  }

  .tp-count {
    font-size: 13px;
    color: var(--text-secondary);
    font-weight: 600;
  }

  .tp-btn-primary {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: var(--radius-sm);
    padding: 6px 14px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
  }
  .tp-btn-primary:hover { opacity: 0.85; }
  .tp-btn-primary:disabled { opacity: 0.4; cursor: default; }

  .tp-btn-secondary {
    background: var(--bg-card);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: var(--radius-sm);
    padding: 6px 14px;
    font-size: 12px;
    cursor: pointer;
    transition: background 0.15s;
  }
  .tp-btn-secondary:hover { background: var(--bg-hover); }

  .tp-empty {
    text-align: center;
    padding: 48px 0;
    color: var(--text-secondary);
  }
  .tp-empty-sub {
    font-size: 12px;
    margin-top: 4px;
    color: var(--text-muted);
  }

  .tp-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .tp-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border-radius: var(--radius-sm);
    transition: background 0.15s;
  }
  .tp-row:hover {
    background: var(--bg-hover);
  }

  .tp-avatar {
    position: relative;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .tp-type-icon {
    font-size: 14px;
  }

  .tp-status-dot {
    position: absolute;
    bottom: -1px;
    right: -1px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    border: 2px solid var(--bg-primary);
  }

  .tp-info {
    flex: 1;
    min-width: 0;
  }

  .tp-name {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tp-meta {
    font-size: 11px;
    color: var(--text-muted);
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 2px;
  }

  .tp-agent-id {
    font-family: var(--font-mono);
  }

  .tp-sep {
    opacity: 0.5;
  }

  .tp-badges {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
  }

  .tp-badge {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border: 1px solid var(--border);
    color: var(--text-secondary);
    background: transparent;
  }

  .tp-remove {
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 14px;
    padding: 4px 8px;
    border-radius: var(--radius-sm);
    opacity: 0;
    transition: all 0.15s;
    flex-shrink: 0;
  }
  .tp-row:hover .tp-remove { opacity: 1; }
  .tp-remove:hover { color: #ef4444; background: rgba(239, 68, 68, 0.1); }

  /* Modal */
  .tp-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .tp-modal {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 24px;
    width: 400px;
    max-width: 90vw;
  }

  .tp-modal h3 {
    margin: 0 0 16px 0;
    font-size: 16px;
    color: var(--text-primary);
  }

  .tp-form-group {
    margin-bottom: 14px;
  }

  .tp-form-group label {
    display: block;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 6px;
  }

  .tp-form-group select {
    width: 100%;
    padding: 8px 10px;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: 13px;
  }

  .tp-empty-modal {
    color: var(--text-secondary);
    font-size: 13px;
    margin: 12px 0;
  }

  .tp-modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 20px;
  }
</style>

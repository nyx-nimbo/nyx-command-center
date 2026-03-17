<script>
  import { push } from 'svelte-spa-router'

  const wails = window['go']?.['main']?.['App']

  let projects = []
  let clients = {}
  let subProjectCounts = {}
  let loading = true
  let search = ''
  let statusFilter = 'all'

  // Modal state
  let modal = false
  let modalMode = 'create'
  let editingProject = { clientId: '', businessUnitId: '', name: '', description: '', status: 'active', stack: '', repoUrl: '', priority: 'medium', startDate: '', dueDate: '', isGroup: false }
  let allClients = []
  let allBusinessUnits = []
  let deleteConfirm = null

  async function loadProjects() {
    try {
      loading = true
      const [allProjects, clientList] = await Promise.all([
        wails.GetAllProjects(),
        wails.GetClients()
      ])
      projects = allProjects || []
      // Build client lookup map
      const cl = clientList || []
      clients = {}
      for (const c of cl) {
        clients[c.id] = c.name
      }
      // Load sub-project counts for groups
      const counts = {}
      for (const p of projects) {
        if (p.isGroup) {
          try { counts[p.id] = await wails.GetSubProjectCount(p.id) } catch { counts[p.id] = 0 }
        }
      }
      subProjectCounts = counts
    } catch (e) {
      console.error('Failed to load projects:', e)
      projects = []
    } finally {
      loading = false
    }
  }

  function openProject(projectId) {
    push(`/project/${projectId}`)
  }

  // Modal
  async function openCreateProject() {
    modal = true
    modalMode = 'create'
    editingProject = { clientId: '', businessUnitId: '', name: '', description: '', status: 'active', stack: '', repoUrl: '', priority: 'medium', startDate: '', dueDate: '', isGroup: false }
    try { allClients = await wails.GetClients() || [] } catch { allClients = [] }
    allBusinessUnits = []
  }

  async function openEditProject(project) {
    modal = true
    modalMode = 'edit'
    editingProject = { ...project }
    try { allClients = await wails.GetClients() || [] } catch { allClients = [] }
    if (editingProject.clientId) {
      try { allBusinessUnits = await wails.GetBusinessUnits(editingProject.clientId) || [] } catch { allBusinessUnits = [] }
    } else {
      allBusinessUnits = []
    }
  }

  async function onProjectClientChange() {
    editingProject.businessUnitId = ''
    if (editingProject.clientId) {
      try { allBusinessUnits = await wails.GetBusinessUnits(editingProject.clientId) || [] } catch { allBusinessUnits = [] }
    } else {
      allBusinessUnits = []
    }
  }

  async function saveProject() {
    try {
      if (modalMode === 'create') {
        await wails.CreateProject(editingProject)
      } else {
        await wails.UpdateProject(editingProject)
      }
      modal = false
      await loadProjects()
    } catch (e) {
      console.error('Failed to save project:', e)
    }
  }

  async function executeDelete() {
    try {
      await wails.DeleteProject(deleteConfirm.id)
      await loadProjects()
    } catch (e) {
      console.error('Delete failed:', e)
    }
    deleteConfirm = null
  }

  function statusColor(status) {
    const colors = { active: '#22c55e', paused: '#f59e0b', completed: '#3b82f6', archived: '#71717a' }
    return colors[status] || '#71717a'
  }

  function priorityColor(priority) {
    const colors = { urgent: '#ef4444', high: '#f97316', medium: '#3b82f6', low: '#71717a' }
    return colors[priority] || '#71717a'
  }

  $: filtered = projects.filter(p => {
    const matchesSearch = !search ||
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      (p.description || '').toLowerCase().includes(search.toLowerCase()) ||
      (p.stack || '').toLowerCase().includes(search.toLowerCase())
    const matchesStatus = statusFilter === 'all' || p.status === statusFilter
    return matchesSearch && matchesStatus
  })

  loadProjects()
</script>

<div class="projects-page">
  <div class="page-header">
    <div class="header-left">
      <h1>Projects</h1>
      <span class="count">{filtered.length} of {projects.length} projects</span>
    </div>
    <div class="header-right">
      <input class="search-input" type="text" placeholder="Search projects..." bind:value={search} />
      <div class="filter-group">
        {#each ['all', 'active', 'paused', 'completed', 'archived'] as s}
          <button
            class="filter-btn"
            class:active={statusFilter === s}
            on:click={() => statusFilter = s}
          >{s}</button>
        {/each}
      </div>
      <button class="btn-primary" on:click={openCreateProject}>+ New Project</button>
    </div>
  </div>

  {#if loading}
    <div class="loading">Loading projects...</div>
  {:else if filtered.length === 0}
    <div class="empty-state">
      <div class="empty-icon">&#9782;</div>
      <p>{search || statusFilter !== 'all' ? 'No projects match your filters' : 'No projects yet'}</p>
      {#if !search && statusFilter === 'all'}
        <button class="btn-primary" on:click={openCreateProject}>Create your first project</button>
      {/if}
    </div>
  {:else}
    <div class="projects-grid">
      {#each filtered as project (project.id)}
        <div class="project-card" on:click={() => openProject(project.id)}>
          <div class="project-top">
            <span class="project-type-icon">{project.isGroup ? '&#128193;' : '&#9776;'}</span>
            <span class="project-name">{project.name}</span>
            <div class="project-badges">
              {#if project.isGroup && subProjectCounts[project.id]}
                <span class="badge sub-count-badge">{subProjectCounts[project.id]} sub</span>
              {/if}
              <span class="badge" style="background: {statusColor(project.status)}20; color: {statusColor(project.status)}">{project.status}</span>
              <span class="badge" style="background: {priorityColor(project.priority)}20; color: {priorityColor(project.priority)}">{project.priority}</span>
            </div>
          </div>
          {#if project.clientId && clients[project.clientId]}
            <div class="project-client">{clients[project.clientId]}</div>
          {/if}
          {#if project.description}<div class="project-desc">{project.description}</div>{/if}
          {#if project.stack}<div class="project-stack">{project.stack}</div>{/if}
          <div class="project-actions-row" on:click|stopPropagation>
            <button class="btn-icon" on:click={() => openEditProject(project)} title="Edit">&#9998;</button>
            <button class="btn-icon btn-danger" on:click={() => { deleteConfirm = project }} title="Delete">&#10005;</button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Create/Edit Modal -->
{#if modal}
  <div class="modal-overlay" on:click={() => modal = false}>
    <div class="modal" on:click|stopPropagation>
      <h2>{modalMode === 'create' ? 'New Project' : 'Edit Project'}</h2>
      <div class="form-group">
        <label>Name *</label>
        <input type="text" bind:value={editingProject.name} placeholder="Project name" />
      </div>
      <div class="form-group">
        <label>Description</label>
        <textarea bind:value={editingProject.description} placeholder="Project description..." rows="3"></textarea>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Client (optional)</label>
          <select bind:value={editingProject.clientId} on:change={onProjectClientChange}>
            <option value="">(No client)</option>
            {#each allClients as c}
              <option value={c.id}>{c.name}</option>
            {/each}
          </select>
        </div>
        <div class="form-group">
          <label>Business Unit (optional)</label>
          <select bind:value={editingProject.businessUnitId} disabled={!editingProject.clientId}>
            <option value="">(None)</option>
            {#each allBusinessUnits as bu}
              <option value={bu.id}>{bu.name}</option>
            {/each}
          </select>
        </div>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Status</label>
          <select bind:value={editingProject.status}>
            <option value="active">Active</option>
            <option value="paused">Paused</option>
            <option value="completed">Completed</option>
            <option value="archived">Archived</option>
          </select>
        </div>
        <div class="form-group">
          <label>Priority</label>
          <select bind:value={editingProject.priority}>
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="urgent">Urgent</option>
          </select>
        </div>
      </div>
      <div class="form-group">
        <label>Stack</label>
        <input type="text" bind:value={editingProject.stack} placeholder="e.g. React Native, Node.js, MongoDB" />
      </div>
      <div class="form-group">
        <label>Repository URL</label>
        <input type="text" bind:value={editingProject.repoUrl} placeholder="https://github.com/..." />
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Start Date</label>
          <input type="date" bind:value={editingProject.startDate} />
        </div>
        <div class="form-group">
          <label>Due Date</label>
          <input type="date" bind:value={editingProject.dueDate} />
        </div>
      </div>
      <div class="form-group">
        <label class="checkbox-label">
          <input type="checkbox" bind:checked={editingProject.isGroup} />
          <span>This is a project group (contains sub-projects)</span>
        </label>
      </div>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => modal = false}>Cancel</button>
        <button class="btn-primary" on:click={saveProject} disabled={!editingProject.name}>Save</button>
      </div>
    </div>
  </div>
{/if}

<!-- Delete Confirm -->
{#if deleteConfirm}
  <div class="modal-overlay" on:click={() => deleteConfirm = null}>
    <div class="modal modal-small" on:click|stopPropagation>
      <h2>Confirm Delete</h2>
      <p>Delete <strong>{deleteConfirm.name}</strong>? This cannot be undone.</p>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => deleteConfirm = null}>Cancel</button>
        <button class="btn-danger-solid" on:click={executeDelete}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .projects-page {
    padding: 24px;
    height: 100%;
    overflow-y: auto;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
    flex-wrap: wrap;
    gap: 12px;
  }

  .header-left {
    display: flex;
    align-items: baseline;
    gap: 12px;
  }

  .header-left h1 {
    font-size: 22px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .count {
    font-size: 13px;
    color: var(--text-muted);
  }

  .header-right {
    display: flex;
    gap: 10px;
    align-items: center;
    flex-wrap: wrap;
  }

  .search-input {
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 8px 12px;
    font-size: 13px;
    width: 220px;
    outline: none;
    transition: border-color var(--transition-fast);
  }

  .search-input:focus {
    border-color: var(--accent);
  }

  .search-input::placeholder {
    color: var(--text-muted);
  }

  .filter-group {
    display: flex;
    gap: 2px;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: 2px;
  }

  .filter-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    padding: 5px 10px;
    font-size: 11px;
    font-weight: 500;
    text-transform: capitalize;
    cursor: pointer;
    border-radius: 4px;
    transition: all var(--transition-fast);
  }

  .filter-btn:hover {
    color: var(--text-secondary);
  }

  .filter-btn.active {
    background: var(--accent-subtle);
    color: var(--accent);
  }

  .loading {
    text-align: center;
    color: var(--text-secondary);
    padding: 48px 0;
  }

  .empty-state {
    text-align: center;
    padding: 64px 0;
    color: var(--text-secondary);
  }

  .empty-icon {
    font-size: 48px;
    color: var(--text-muted);
    margin-bottom: 16px;
  }

  .empty-state p {
    margin-bottom: 16px;
  }

  .projects-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 12px;
  }

  .project-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: 16px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .project-card:hover {
    border-color: var(--accent);
    background: var(--bg-card-hover);
  }

  .project-top {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 8px;
    margin-bottom: 6px;
  }

  .project-type-icon {
    font-size: 16px;
    flex-shrink: 0;
  }

  .project-name {
    font-weight: 600;
    font-size: 14px;
    color: var(--text-primary);
    flex: 1;
  }

  .sub-count-badge {
    background: rgba(139, 92, 246, 0.15);
    color: #8b5cf6;
  }

  .project-badges {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }

  .badge {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    text-transform: uppercase;
  }

  .project-client {
    font-size: 12px;
    color: var(--accent);
    margin-bottom: 4px;
  }

  .project-desc {
    font-size: 12px;
    color: var(--text-secondary);
    margin-bottom: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .project-stack {
    font-size: 11px;
    color: var(--text-muted);
    margin-bottom: 4px;
  }

  .project-actions-row {
    display: flex;
    gap: 4px;
    opacity: 0;
    transition: opacity var(--transition-fast);
    margin-top: 8px;
    justify-content: flex-end;
  }

  .project-card:hover .project-actions-row {
    opacity: 1;
  }

  /* Buttons */
  .btn-primary {
    background: var(--accent);
    color: white;
    border: none;
    border-radius: var(--radius-sm);
    padding: 8px 16px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .btn-primary:hover {
    background: var(--accent-hover);
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-secondary {
    background: var(--bg-card);
    color: var(--text-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 8px 16px;
    font-size: 13px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .btn-secondary:hover {
    background: var(--bg-card-hover);
    color: var(--text-primary);
  }

  .btn-icon {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px 6px;
    border-radius: 4px;
    font-size: 14px;
    transition: all var(--transition-fast);
  }

  .btn-icon:hover {
    background: var(--bg-card-hover);
    color: var(--text-primary);
  }

  .btn-icon.btn-danger:hover {
    color: var(--error);
  }

  .btn-danger-solid {
    background: var(--error);
    color: white;
    border: none;
    border-radius: var(--radius-sm);
    padding: 8px 16px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity var(--transition-fast);
  }

  .btn-danger-solid:hover {
    opacity: 0.9;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 24px;
    width: 480px;
    max-width: 90vw;
    max-height: 85vh;
    overflow-y: auto;
  }

  .modal-small {
    width: 360px;
  }

  .modal h2 {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 16px;
  }

  .modal p {
    font-size: 13px;
    color: var(--text-secondary);
    margin-bottom: 16px;
  }

  .form-group {
    margin-bottom: 12px;
  }

  .form-group label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 4px;
  }

  .form-group input,
  .form-group textarea,
  .form-group select {
    width: 100%;
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 8px 10px;
    font-size: 13px;
    font-family: inherit;
    outline: none;
    transition: border-color var(--transition-fast);
  }

  .form-group input:focus,
  .form-group textarea:focus,
  .form-group select:focus {
    border-color: var(--accent);
  }

  .form-group select {
    cursor: pointer;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 16px;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .checkbox-label input[type="checkbox"] {
    width: auto;
    accent-color: var(--accent);
    cursor: pointer;
  }
</style>

<script>
  import { push } from 'svelte-spa-router'

  const wails = window['go']?.['main']?.['App']

  let clients = []
  let businessUnits = {}
  let projects = {}
  let expandedClient = null
  let expandedBU = null
  let loading = true
  let search = ''
  let unassignedProjects = []

  // Modal state
  let modal = null // 'client' | 'bu' | 'project'
  let modalMode = 'create' // 'create' | 'edit'
  let editingClient = { name: '', contactName: '', contactEmail: '', phone: '', notes: '' }
  let editingBU = { clientId: '', name: '', rfc: '', address: '', notes: '' }
  let editingProject = { clientId: '', businessUnitId: '', name: '', description: '', status: 'active', stack: '', repoUrl: '', priority: 'medium', startDate: '', dueDate: '' }
  let deleteConfirm = null
  let allClients = []
  let allBusinessUnits = []

  async function loadClients() {
    try {
      loading = true
      clients = await wails.GetClients() || []
      // Load unassigned projects (no clientId)
      const allProjs = await wails.GetProjects('', '') || []
      unassignedProjects = allProjs.filter(p => !p.clientId)
    } catch (e) {
      console.error('Failed to load clients:', e)
      clients = []
    } finally {
      loading = false
    }
  }

  async function loadBusinessUnits(clientId) {
    try {
      const units = await wails.GetBusinessUnits(clientId) || []
      businessUnits = { ...businessUnits, [clientId]: units }
    } catch (e) {
      console.error('Failed to load business units:', e)
      businessUnits = { ...businessUnits, [clientId]: [] }
    }
  }

  async function loadProjects(clientId, buId) {
    try {
      const projs = await wails.GetProjects(clientId, buId) || []
      projects = { ...projects, [buId]: projs }
    } catch (e) {
      console.error('Failed to load projects:', e)
      projects = { ...projects, [buId]: [] }
    }
  }

  function toggleClient(clientId) {
    if (expandedClient === clientId) {
      expandedClient = null
      expandedBU = null
    } else {
      expandedClient = clientId
      expandedBU = null
      loadBusinessUnits(clientId)
    }
  }

  function toggleBU(clientId, buId) {
    if (expandedBU === buId) {
      expandedBU = null
    } else {
      expandedBU = buId
      loadProjects(clientId, buId)
    }
  }

  // Client modal
  function openCreateClient() {
    modal = 'client'
    modalMode = 'create'
    editingClient = { name: '', contactName: '', contactEmail: '', phone: '', notes: '' }
  }

  function openEditClient(client) {
    modal = 'client'
    modalMode = 'edit'
    editingClient = { ...client }
  }

  async function saveClient() {
    try {
      if (modalMode === 'create') {
        await wails.CreateClient(editingClient)
      } else {
        await wails.UpdateClient(editingClient)
      }
      modal = null
      await loadClients()
    } catch (e) {
      console.error('Failed to save client:', e)
    }
  }

  async function confirmDeleteClient(client) {
    deleteConfirm = { type: 'client', item: client }
  }

  async function executeDelete() {
    try {
      if (deleteConfirm.type === 'client') {
        await wails.DeleteClient(deleteConfirm.item.id)
        expandedClient = null
        await loadClients()
      } else if (deleteConfirm.type === 'bu') {
        await wails.DeleteBusinessUnit(deleteConfirm.item.id)
        expandedBU = null
        if (expandedClient) await loadBusinessUnits(expandedClient)
      } else if (deleteConfirm.type === 'project') {
        await wails.DeleteProject(deleteConfirm.item.id)
        if (expandedBU) await loadProjects(expandedClient, expandedBU)
      }
    } catch (e) {
      console.error('Delete failed:', e)
    }
    deleteConfirm = null
  }

  // BU modal
  function openCreateBU(clientId) {
    modal = 'bu'
    modalMode = 'create'
    editingBU = { clientId, name: '', rfc: '', address: '', notes: '' }
  }

  function openEditBU(bu) {
    modal = 'bu'
    modalMode = 'edit'
    editingBU = { ...bu }
  }

  async function saveBU() {
    try {
      if (modalMode === 'create') {
        await wails.CreateBusinessUnit(editingBU)
      } else {
        await wails.UpdateBusinessUnit(editingBU)
      }
      modal = null
      if (editingBU.clientId) await loadBusinessUnits(editingBU.clientId)
    } catch (e) {
      console.error('Failed to save BU:', e)
    }
  }

  // Project modal
  async function openCreateProject(clientId, buId) {
    modal = 'project'
    modalMode = 'create'
    editingProject = { clientId: clientId || '', businessUnitId: buId || '', name: '', description: '', status: 'active', stack: '', repoUrl: '', priority: 'medium', startDate: '', dueDate: '' }
    try { allClients = await wails.GetClients() || [] } catch { allClients = [] }
    if (clientId) {
      try { allBusinessUnits = await wails.GetBusinessUnits(clientId) || [] } catch { allBusinessUnits = [] }
    } else {
      allBusinessUnits = []
    }
  }

  async function openEditProject(project) {
    modal = 'project'
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
      modal = null
      if (editingProject.businessUnitId) await loadProjects(editingProject.clientId, editingProject.businessUnitId)
      await loadClients()
    } catch (e) {
      console.error('Failed to save project:', e)
    }
  }

  function openProject(projectId) {
    push(`/project/${projectId}`)
  }

  function statusColor(status) {
    const colors = { active: '#22c55e', paused: '#f59e0b', completed: '#3b82f6', archived: '#71717a' }
    return colors[status] || '#71717a'
  }

  function priorityColor(priority) {
    const colors = { urgent: '#ef4444', high: '#f97316', medium: '#3b82f6', low: '#71717a' }
    return colors[priority] || '#71717a'
  }

  $: filteredClients = clients.filter(c =>
    c.name.toLowerCase().includes(search.toLowerCase()) ||
    c.contactName.toLowerCase().includes(search.toLowerCase()) ||
    c.contactEmail.toLowerCase().includes(search.toLowerCase())
  )

  loadClients()
</script>

<div class="clients-page">
  <div class="page-header">
    <div class="header-left">
      <h1>Clients</h1>
      <span class="count">{clients.length} clients</span>
    </div>
    <div class="header-right">
      <input class="search-input" type="text" placeholder="Search clients..." bind:value={search} />
      <button class="btn-secondary" on:click={() => openCreateProject('', '')}>+ New Project</button>
      <button class="btn-primary" on:click={openCreateClient}>+ New Client</button>
    </div>
  </div>

  {#if loading}
    <div class="loading">Loading clients...</div>
  {:else if filteredClients.length === 0}
    <div class="empty-state">
      <div class="empty-icon">◈</div>
      <p>No clients found</p>
      <button class="btn-primary" on:click={openCreateClient}>Add your first client</button>
    </div>
  {:else}
    <div class="clients-list">
      {#each filteredClients as client (client.id)}
        <div class="client-card" class:expanded={expandedClient === client.id}>
          <div class="client-header" on:click={() => toggleClient(client.id)}>
            <div class="client-info">
              <span class="expand-icon">{expandedClient === client.id ? '▾' : '▸'}</span>
              <div>
                <div class="client-name">{client.name}</div>
                <div class="client-contact">
                  {#if client.contactName}{client.contactName}{/if}
                  {#if client.contactEmail} · {client.contactEmail}{/if}
                </div>
              </div>
            </div>
            <div class="client-actions" on:click|stopPropagation>
              <button class="btn-icon" on:click={() => openEditClient(client)} title="Edit">✎</button>
              <button class="btn-icon btn-danger" on:click={() => confirmDeleteClient(client)} title="Delete">✕</button>
            </div>
          </div>

          {#if expandedClient === client.id}
            <div class="client-body">
              <div class="section-header">
                <span class="section-title">Business Units</span>
                <button class="btn-small" on:click={() => openCreateBU(client.id)}>+ Add</button>
              </div>

              {#if !businessUnits[client.id] || businessUnits[client.id].length === 0}
                <div class="empty-sub">No business units yet</div>
              {:else}
                {#each businessUnits[client.id] as bu (bu.id)}
                  <div class="bu-card" class:expanded={expandedBU === bu.id}>
                    <div class="bu-header" on:click={() => toggleBU(client.id, bu.id)}>
                      <div class="bu-info">
                        <span class="expand-icon">{expandedBU === bu.id ? '▾' : '▸'}</span>
                        <div>
                          <div class="bu-name">{bu.name}</div>
                          {#if bu.rfc}<div class="bu-rfc">RFC: {bu.rfc}</div>{/if}
                        </div>
                      </div>
                      <div class="bu-actions" on:click|stopPropagation>
                        <button class="btn-icon" on:click={() => openEditBU(bu)} title="Edit">✎</button>
                        <button class="btn-icon btn-danger" on:click={() => { deleteConfirm = { type: 'bu', item: bu } }} title="Delete">✕</button>
                      </div>
                    </div>

                    {#if expandedBU === bu.id}
                      <div class="bu-body">
                        <div class="section-header">
                          <span class="section-title">Projects</span>
                          <button class="btn-small" on:click={() => openCreateProject(client.id, bu.id)}>+ Add</button>
                        </div>

                        {#if !projects[bu.id] || projects[bu.id].length === 0}
                          <div class="empty-sub">No projects yet</div>
                        {:else}
                          <div class="projects-grid">
                            {#each projects[bu.id] as project (project.id)}
                              <div class="project-card" on:click={() => openProject(project.id)}>
                                <div class="project-top">
                                  <span class="project-name">{project.name}</span>
                                  <div class="project-badges">
                                    <span class="badge" style="background: {statusColor(project.status)}20; color: {statusColor(project.status)}">{project.status}</span>
                                    <span class="badge" style="background: {priorityColor(project.priority)}20; color: {priorityColor(project.priority)}">{project.priority}</span>
                                  </div>
                                </div>
                                {#if project.description}<div class="project-desc">{project.description}</div>{/if}
                                {#if project.stack}<div class="project-stack">{project.stack}</div>{/if}
                                <div class="project-actions-row" on:click|stopPropagation>
                                  <button class="btn-icon" on:click={() => openEditProject(project)} title="Edit">✎</button>
                                  <button class="btn-icon btn-danger" on:click={() => { deleteConfirm = { type: 'project', item: project } }} title="Delete">✕</button>
                                </div>
                              </div>
                            {/each}
                          </div>
                        {/if}
                      </div>
                    {/if}
                  </div>
                {/each}
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>

    <!-- Unassigned Projects -->
    {#if unassignedProjects.length > 0}
      <div class="unassigned-section">
        <div class="section-header">
          <span class="section-title">Unassigned Projects (No Client)</span>
        </div>
        <div class="projects-grid">
          {#each unassignedProjects as project (project.id)}
            <div class="project-card" on:click={() => openProject(project.id)}>
              <div class="project-top">
                <span class="project-name">{project.name}</span>
                <div class="project-badges">
                  <span class="badge" style="background: {statusColor(project.status)}20; color: {statusColor(project.status)}">{project.status}</span>
                  <span class="badge" style="background: {priorityColor(project.priority)}20; color: {priorityColor(project.priority)}">{project.priority}</span>
                </div>
              </div>
              {#if project.description}<div class="project-desc">{project.description}</div>{/if}
              {#if project.stack}<div class="project-stack">{project.stack}</div>{/if}
              <div class="project-actions-row" on:click|stopPropagation>
                <button class="btn-icon" on:click={() => openEditProject(project)} title="Edit">✎</button>
                <button class="btn-icon btn-danger" on:click={() => { deleteConfirm = { type: 'project', item: project } }} title="Delete">✕</button>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- Modals -->
{#if modal === 'client'}
  <div class="modal-overlay" on:click={() => modal = null}>
    <div class="modal" on:click|stopPropagation>
      <h2>{modalMode === 'create' ? 'New Client' : 'Edit Client'}</h2>
      <div class="form-group">
        <label>Name *</label>
        <input type="text" bind:value={editingClient.name} placeholder="Company name" />
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Contact Name</label>
          <input type="text" bind:value={editingClient.contactName} placeholder="Contact person" />
        </div>
        <div class="form-group">
          <label>Contact Email</label>
          <input type="email" bind:value={editingClient.contactEmail} placeholder="email@example.com" />
        </div>
      </div>
      <div class="form-group">
        <label>Phone</label>
        <input type="text" bind:value={editingClient.phone} placeholder="Phone number" />
      </div>
      <div class="form-group">
        <label>Notes</label>
        <textarea bind:value={editingClient.notes} placeholder="Additional notes..." rows="3"></textarea>
      </div>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => modal = null}>Cancel</button>
        <button class="btn-primary" on:click={saveClient} disabled={!editingClient.name}>Save</button>
      </div>
    </div>
  </div>
{/if}

{#if modal === 'bu'}
  <div class="modal-overlay" on:click={() => modal = null}>
    <div class="modal" on:click|stopPropagation>
      <h2>{modalMode === 'create' ? 'New Business Unit' : 'Edit Business Unit'}</h2>
      <div class="form-group">
        <label>Name *</label>
        <input type="text" bind:value={editingBU.name} placeholder="Business unit name" />
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>RFC</label>
          <input type="text" bind:value={editingBU.rfc} placeholder="RFC identifier" />
        </div>
        <div class="form-group">
          <label>Address</label>
          <input type="text" bind:value={editingBU.address} placeholder="Address" />
        </div>
      </div>
      <div class="form-group">
        <label>Notes</label>
        <textarea bind:value={editingBU.notes} placeholder="Additional notes..." rows="3"></textarea>
      </div>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => modal = null}>Cancel</button>
        <button class="btn-primary" on:click={saveBU} disabled={!editingBU.name}>Save</button>
      </div>
    </div>
  </div>
{/if}

{#if modal === 'project'}
  <div class="modal-overlay" on:click={() => modal = null}>
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
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => modal = null}>Cancel</button>
        <button class="btn-primary" on:click={saveProject} disabled={!editingProject.name}>Save</button>
      </div>
    </div>
  </div>
{/if}

{#if deleteConfirm}
  <div class="modal-overlay" on:click={() => deleteConfirm = null}>
    <div class="modal modal-small" on:click|stopPropagation>
      <h2>Confirm Delete</h2>
      <p>Delete <strong>{deleteConfirm.item.name}</strong>? This cannot be undone.</p>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => deleteConfirm = null}>Cancel</button>
        <button class="btn-danger-solid" on:click={executeDelete}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .clients-page {
    padding: 24px;
    height: 100%;
    overflow-y: auto;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
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

  .empty-sub {
    font-size: 13px;
    color: var(--text-muted);
    padding: 12px 16px;
  }

  .clients-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .client-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    overflow: hidden;
    transition: border-color var(--transition-fast);
  }

  .client-card:hover {
    border-color: var(--border);
  }

  .client-card.expanded {
    border-color: var(--accent);
  }

  .client-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 16px;
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .client-header:hover {
    background: var(--bg-card-hover);
  }

  .client-info {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .expand-icon {
    font-size: 12px;
    color: var(--text-muted);
    width: 16px;
  }

  .client-name {
    font-weight: 600;
    color: var(--text-primary);
    font-size: 14px;
  }

  .client-contact {
    font-size: 12px;
    color: var(--text-secondary);
    margin-top: 2px;
  }

  .client-actions, .bu-actions, .project-actions-row {
    display: flex;
    gap: 4px;
    opacity: 0;
    transition: opacity var(--transition-fast);
  }

  .client-header:hover .client-actions,
  .bu-header:hover .bu-actions,
  .project-card:hover .project-actions-row {
    opacity: 1;
  }

  .client-body {
    border-top: 1px solid var(--border-subtle);
    padding: 12px 16px 16px;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .section-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .bu-card {
    background: var(--bg-primary);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    margin-bottom: 6px;
    overflow: hidden;
  }

  .bu-card.expanded {
    border-color: var(--accent);
  }

  .bu-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .bu-header:hover {
    background: var(--bg-card-hover);
  }

  .bu-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .bu-name {
    font-weight: 500;
    color: var(--text-primary);
    font-size: 13px;
  }

  .bu-rfc {
    font-size: 11px;
    color: var(--text-muted);
  }

  .bu-body {
    border-top: 1px solid var(--border-subtle);
    padding: 10px 12px 12px;
  }

  .projects-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
    gap: 8px;
  }

  .project-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: 12px;
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

  .project-name {
    font-weight: 600;
    font-size: 13px;
    color: var(--text-primary);
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
    margin-top: 6px;
    justify-content: flex-end;
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

  .btn-small {
    background: transparent;
    color: var(--accent);
    border: 1px solid var(--accent);
    border-radius: var(--radius-sm);
    padding: 3px 10px;
    font-size: 11px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .btn-small:hover {
    background: var(--accent-subtle);
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
</style>

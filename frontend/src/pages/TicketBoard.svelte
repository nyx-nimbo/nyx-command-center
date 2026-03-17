<script>
  import { onMount } from 'svelte'

  export let projectId = ''

  const wails = window['go']?.['main']?.['App']

  let tickets = []
  let epics = []
  let stats = null
  let loading = true

  // Filters
  let filterEpic = ''
  let filterPriority = ''
  let filterType = ''
  let filterAssignee = ''

  // Columns
  const columns = [
    { key: 'draft', label: 'Draft', color: '#71717a', icon: '○' },
    { key: 'ready', label: 'Ready', color: '#8b5cf6', icon: '◎' },
    { key: 'in_progress', label: 'In Progress', color: '#3b82f6', icon: '◉' },
    { key: 'review', label: 'Review', color: '#f59e0b', icon: '◈' },
    { key: 'done', label: 'Done', color: '#22c55e', icon: '●' },
  ]

  // Ticket modal
  let ticketModal = false
  let ticketMode = 'create'
  let editingTicket = emptyTicket()
  let tagInput = ''
  let acInput = ''

  // Epic modal
  let epicModal = false
  let epicMode = 'create'
  let editingEpic = emptyEpic()

  // Detail modal
  let detailModal = false
  let detailTicket = null

  // Bulk select
  let bulkMode = false
  let selectedIds = new Set()

  // Delete confirm
  let deleteConfirm = null

  // Generate modal
  let generateModal = false
  let generateDescription = ''
  let generateCreateEpic = true
  let generateStatus = 'draft'
  let generating = false
  let generateNotification = null

  // Drag state
  let dragTicketId = null
  let dragOverColumn = null

  // Project agents for assignee dropdown
  let projectAgents = []

  function emptyTicket() {
    return {
      projectId: projectId,
      epicId: '',
      title: '',
      description: '',
      scope: '',
      acceptanceCriteria: [],
      technicalNotes: '',
      type: 'feature',
      status: 'draft',
      priority: 'medium',
      estimate: '',
      storyPoints: 0,
      assignedTo: '',
      tags: [],
      order: 0,
    }
  }

  function emptyEpic() {
    return {
      projectId: projectId,
      title: '',
      description: '',
      status: 'open',
    }
  }

  function priorityColor(p) {
    return { critical: '#ef4444', high: '#f97316', medium: '#3b82f6', low: '#71717a' }[p] || '#71717a'
  }

  function priorityLabel(p) {
    return { critical: 'CRIT', high: 'HIGH', medium: 'MED', low: 'LOW' }[p] || p
  }

  function typeIcon(t) {
    return { feature: '★', bug: '⊘', chore: '⚙', spike: '⚡' }[t] || '◦'
  }

  function agentName(agentId) {
    if (!agentId) return ''
    const ag = projectAgents.find(a => a.agentId === agentId)
    return ag ? ag.name : agentId
  }

  function epicName(epicId) {
    if (!epicId) return ''
    const e = epics.find(ep => ep.id === epicId)
    return e ? e.title : ''
  }

  function ticketsByColumn(colKey) {
    let filtered = tickets.filter(t => t.status === colKey)
    if (filterEpic) filtered = filtered.filter(t => t.epicId === filterEpic)
    if (filterPriority) filtered = filtered.filter(t => t.priority === filterPriority)
    if (filterType) filtered = filtered.filter(t => t.type === filterType)
    if (filterAssignee) filtered = filtered.filter(t => t.assignedTo === filterAssignee)
    return filtered
  }

  $: allAssignees = [...new Set(tickets.map(t => t.assignedTo).filter(Boolean))].sort()
  $: filteredTotal = columns.reduce((sum, col) => sum + ticketsByColumn(col.key).length, 0)
  $: hasFilters = filterEpic || filterPriority || filterType || filterAssignee

  // --- Data loading ---
  async function loadTickets() {
    try { tickets = await wails.GetTicketsByProject(projectId) || [] } catch (e) { tickets = [] }
  }

  async function loadEpics() {
    try { epics = await wails.GetEpicsByProject(projectId) || [] } catch (e) { epics = [] }
  }

  async function loadStats() {
    try { stats = await wails.GetTicketStats(projectId) } catch (e) { stats = null }
  }

  async function loadProjectAgents() {
    try { projectAgents = await wails.GetProjectAssignments(projectId) || [] } catch (e) { projectAgents = [] }
  }

  async function loadAll() {
    loading = true
    await Promise.all([loadTickets(), loadEpics(), loadStats(), loadProjectAgents()])
    loading = false
  }

  onMount(() => { if (projectId) loadAll() })

  // Reactively reload when projectId changes
  $: if (projectId) loadAll()

  // --- Ticket CRUD ---
  function openCreateTicket(status) {
    ticketModal = true
    ticketMode = 'create'
    editingTicket = { ...emptyTicket(), projectId, status }
    tagInput = ''
    acInput = ''
  }

  function openEditTicket(ticket) {
    ticketModal = true
    ticketMode = 'edit'
    editingTicket = { ...ticket, tags: ticket.tags || [], acceptanceCriteria: ticket.acceptanceCriteria || [] }
    tagInput = ''
    acInput = ''
  }

  function openDetail(ticket) {
    detailTicket = { ...ticket }
    detailModal = true
  }

  function editFromDetail() {
    detailModal = false
    openEditTicket(detailTicket)
  }

  async function saveTicket() {
    try {
      if (ticketMode === 'create') await wails.CreateTicket(editingTicket)
      else await wails.UpdateTicket(editingTicket)
      ticketModal = false
      await loadAll()
    } catch (e) { console.error('Failed to save ticket:', e) }
  }

  async function deleteTicketConfirmed() {
    if (!deleteConfirm) return
    try {
      await wails.DeleteTicket(deleteConfirm.id)
      deleteConfirm = null
      detailModal = false
      await loadAll()
    } catch (e) { console.error('Failed to delete ticket:', e) }
  }

  async function moveTicket(ticketId, newStatus) {
    try {
      await wails.MoveTicket(ticketId, newStatus)
      await loadAll()
    } catch (e) { console.error('Failed to move ticket:', e) }
  }

  // --- Bulk ---
  function toggleBulk() {
    bulkMode = !bulkMode
    if (!bulkMode) selectedIds = new Set()
  }

  function toggleSelect(id) {
    if (selectedIds.has(id)) {
      selectedIds.delete(id)
    } else {
      selectedIds.add(id)
    }
    selectedIds = new Set(selectedIds)
  }

  async function bulkApprove() {
    const ids = Array.from(selectedIds)
    if (ids.length === 0) return
    try {
      await wails.BulkUpdateTicketStatus(ids, 'ready')
      selectedIds = new Set()
      bulkMode = false
      await loadAll()
    } catch (e) { console.error('Bulk approve failed:', e) }
  }

  // --- Tags ---
  function addTag() {
    const tag = tagInput.trim()
    if (tag && !editingTicket.tags.includes(tag)) editingTicket.tags = [...editingTicket.tags, tag]
    tagInput = ''
  }
  function removeTag(tag) { editingTicket.tags = editingTicket.tags.filter(t => t !== tag) }
  function handleTagKey(e) { if (e.key === 'Enter') { e.preventDefault(); addTag() } }

  // --- Acceptance Criteria ---
  function addAC() {
    const ac = acInput.trim()
    if (ac) editingTicket.acceptanceCriteria = [...editingTicket.acceptanceCriteria, ac]
    acInput = ''
  }
  function removeAC(idx) { editingTicket.acceptanceCriteria = editingTicket.acceptanceCriteria.filter((_, i) => i !== idx) }
  function handleACKey(e) { if (e.key === 'Enter') { e.preventDefault(); addAC() } }

  // --- Epic CRUD ---
  function openCreateEpic() {
    epicModal = true
    epicMode = 'create'
    editingEpic = { ...emptyEpic(), projectId }
  }

  function openEditEpic(epic) {
    epicModal = true
    epicMode = 'edit'
    editingEpic = { ...epic }
  }

  async function saveEpic() {
    try {
      if (epicMode === 'create') await wails.CreateEpic(editingEpic)
      else await wails.UpdateEpic(editingEpic)
      epicModal = false
      await loadEpics()
    } catch (e) { console.error('Failed to save epic:', e) }
  }

  async function deleteEpic(id) {
    try {
      await wails.DeleteEpic(id)
      await loadAll()
    } catch (e) { console.error('Failed to delete epic:', e) }
  }

  // --- Drag & Drop ---
  function onDragStart(e, ticketId) {
    dragTicketId = ticketId
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', ticketId)
  }

  function onDragOver(e, colKey) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    dragOverColumn = colKey
  }

  function onDragLeave() {
    dragOverColumn = null
  }

  async function onDrop(e, colKey) {
    e.preventDefault()
    dragOverColumn = null
    if (!dragTicketId) return
    const ticket = tickets.find(t => t.id === dragTicketId)
    if (ticket && ticket.status !== colKey) {
      await moveTicket(dragTicketId, colKey)
    }
    dragTicketId = null
  }

  function onDragEnd() {
    dragTicketId = null
    dragOverColumn = null
  }

  // --- AI Generate ---
  function openGenerateModal() {
    generateModal = true
    generateDescription = ''
    generateCreateEpic = true
    generateStatus = 'draft'
    generating = false
  }

  async function generateTickets() {
    if (!generateDescription.trim() || generating) return
    generating = true
    try {
      const result = await wails.GenerateTickets({
        projectId: projectId,
        description: generateDescription,
        createEpic: generateCreateEpic,
        status: generateStatus,
      })
      generateModal = false
      generating = false
      const count = result.tickets ? result.tickets.length : 0
      const epicMsg = result.epic ? ` + 1 epic` : ''
      generateNotification = `${count} tickets${epicMsg} created`
      setTimeout(() => { generateNotification = null }, 4000)
      await loadAll()
    } catch (e) {
      generating = false
      console.error('Generate tickets failed:', e)
      generateNotification = 'Generation failed: ' + (e.message || e)
      setTimeout(() => { generateNotification = null }, 5000)
    }
  }

  // Listen for Wails events
  onMount(() => {
    if (window.runtime) {
      window.runtime.EventsOn('tickets:generate-error', (msg) => {
        generating = false
        generateNotification = 'Error: ' + msg
        setTimeout(() => { generateNotification = null }, 5000)
      })
    }
  })

  // --- Filters ---
  function clearFilters() {
    filterEpic = ''
    filterPriority = ''
    filterType = ''
    filterAssignee = ''
  }
</script>

<div class="ticket-board">
  {#if loading}
    <div class="tb-loading">Loading tickets...</div>
  {:else}
    <!-- Stats Bar -->
    {#if stats}
      <div class="tb-stats">
        <div class="tb-stat"><span class="tb-stat-num">{stats.total}</span> Total</div>
        <div class="tb-stat"><span class="tb-stat-num" style="color: #71717a">{stats.draft}</span> Draft</div>
        <div class="tb-stat"><span class="tb-stat-num" style="color: #8b5cf6">{stats.ready}</span> Ready</div>
        <div class="tb-stat"><span class="tb-stat-num" style="color: #3b82f6">{stats.inProgress}</span> Active</div>
        <div class="tb-stat"><span class="tb-stat-num" style="color: #f59e0b">{stats.review}</span> Review</div>
        <div class="tb-stat"><span class="tb-stat-num" style="color: #22c55e">{stats.done}</span> Done</div>
        {#if stats.total > 0}
          <div class="tb-progress">
            <div class="tb-progress-fill" style="width: {(stats.done / stats.total * 100)}%"></div>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Toolbar -->
    <div class="tb-toolbar">
      <div class="tb-toolbar-left">
        <button class="tb-btn-primary" on:click={() => openCreateTicket('draft')}>+ New Ticket</button>
        <button class="tb-btn-generate" on:click={openGenerateModal}>&#10024; Generate</button>
        <button class="tb-btn-secondary" on:click={openCreateEpic}>+ Epic</button>
        <button class="tb-btn-secondary" class:active={bulkMode} on:click={toggleBulk}>
          {bulkMode ? 'Cancel Bulk' : 'Bulk Select'}
        </button>
        {#if bulkMode && selectedIds.size > 0}
          <button class="tb-btn-approve" on:click={bulkApprove}>Approve {selectedIds.size} → Ready</button>
        {/if}
      </div>
      <div class="tb-toolbar-right">
        {#if epics.length > 0}
          <select class="tb-filter" bind:value={filterEpic}>
            <option value="">All Epics</option>
            {#each epics as ep}
              <option value={ep.id}>{ep.code} {ep.title}</option>
            {/each}
          </select>
        {/if}
        <select class="tb-filter" bind:value={filterPriority}>
          <option value="">All Priority</option>
          <option value="critical">Critical</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
        <select class="tb-filter" bind:value={filterType}>
          <option value="">All Types</option>
          <option value="feature">Feature</option>
          <option value="bug">Bug</option>
          <option value="chore">Chore</option>
          <option value="spike">Spike</option>
        </select>
        {#if allAssignees.length > 0}
          <select class="tb-filter" bind:value={filterAssignee}>
            <option value="">All Assignees</option>
            {#each allAssignees as a}
              <option value={a}>{a}</option>
            {/each}
          </select>
        {/if}
        {#if hasFilters}
          <button class="tb-btn-clear" on:click={clearFilters}>Clear</button>
        {/if}
      </div>
    </div>

    <!-- Epics bar (collapsible) -->
    {#if epics.length > 0}
      <div class="tb-epics-bar">
        {#each epics as ep (ep.id)}
          <div class="tb-epic-chip" class:closed={ep.status === 'closed'} on:click={() => openEditEpic(ep)}>
            <span class="tb-epic-code">{ep.code}</span>
            <span class="tb-epic-title">{ep.title}</span>
            <span class="tb-epic-status" style="color: {ep.status === 'open' ? '#22c55e' : '#71717a'}">{ep.status}</span>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Kanban Columns -->
    <div class="tb-kanban">
      {#each columns as col (col.key)}
        <div
          class="tb-column"
          class:drag-over={dragOverColumn === col.key}
          on:dragover={(e) => onDragOver(e, col.key)}
          on:dragleave={onDragLeave}
          on:drop={(e) => onDrop(e, col.key)}
        >
          <div class="tb-col-header">
            <div class="tb-col-title">
              <span class="tb-col-dot" style="background: {col.color}"></span>
              {col.label}
              <span class="tb-col-count">{ticketsByColumn(col.key).length}</span>
            </div>
            <button class="tb-btn-add" on:click={() => openCreateTicket(col.key)} title="Add ticket">+</button>
          </div>
          <div class="tb-col-body">
            {#each ticketsByColumn(col.key) as ticket (ticket.id)}
              <div
                class="tb-card"
                class:selected={selectedIds.has(ticket.id)}
                class:dragging={dragTicketId === ticket.id}
                draggable="true"
                on:dragstart={(e) => onDragStart(e, ticket.id)}
                on:dragend={onDragEnd}
                on:click={() => bulkMode ? toggleSelect(ticket.id) : openDetail(ticket)}
              >
                {#if bulkMode && col.key === 'draft'}
                  <div class="tb-card-check">
                    <input type="checkbox" checked={selectedIds.has(ticket.id)} on:click|stopPropagation={() => toggleSelect(ticket.id)} />
                  </div>
                {/if}
                <div class="tb-card-top">
                  <span class="tb-card-code">{ticket.code}</span>
                  <div class="tb-card-badges">
                    <span class="tb-badge-priority" style="background: {priorityColor(ticket.priority)}20; color: {priorityColor(ticket.priority)}">{priorityLabel(ticket.priority)}</span>
                    {#if ticket.estimate}
                      <span class="tb-badge-estimate">{ticket.estimate}</span>
                    {/if}
                  </div>
                </div>
                <div class="tb-card-title">
                  <span class="tb-type-icon" title={ticket.type}>{typeIcon(ticket.type)}</span>
                  {ticket.title}
                </div>
                {#if epicName(ticket.epicId)}
                  <div class="tb-card-epic">{epicName(ticket.epicId)}</div>
                {/if}
                {#if ticket.tags && ticket.tags.length > 0}
                  <div class="tb-card-tags">
                    {#each ticket.tags.slice(0, 3) as tag}<span class="tb-tag">{tag}</span>{/each}
                    {#if ticket.tags.length > 3}<span class="tb-tag tb-tag-more">+{ticket.tags.length - 3}</span>{/if}
                  </div>
                {/if}
                <div class="tb-card-footer">
                  {#if ticket.assignedTo}
                    <span class="tb-assignee">{agentName(ticket.assignedTo)}</span>
                  {:else}
                    <span class="tb-unassigned">Unassigned</span>
                  {/if}
                  {#if ticket.storyPoints}
                    <span class="tb-points">{ticket.storyPoints}pt</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Detail Modal -->
{#if detailModal && detailTicket}
  <div class="modal-overlay" on:click={() => detailModal = false}>
    <div class="modal modal-lg" on:click|stopPropagation>
      <div class="detail-header">
        <div class="detail-header-left">
          <span class="detail-code">{detailTicket.code}</span>
          <span class="tb-type-icon" title={detailTicket.type}>{typeIcon(detailTicket.type)}</span>
          <span class="tb-badge-priority" style="background: {priorityColor(detailTicket.priority)}20; color: {priorityColor(detailTicket.priority)}">{priorityLabel(detailTicket.priority)}</span>
          {#if detailTicket.estimate}
            <span class="tb-badge-estimate">{detailTicket.estimate}</span>
          {/if}
          {#if detailTicket.storyPoints}
            <span class="tb-points">{detailTicket.storyPoints}pt</span>
          {/if}
        </div>
        <div class="detail-header-right">
          <button class="tb-btn-secondary" on:click={editFromDetail}>Edit</button>
          <button class="tb-btn-danger" on:click={() => deleteConfirm = detailTicket}>Delete</button>
        </div>
      </div>
      <h2 class="detail-title">{detailTicket.title}</h2>

      <!-- Status quick-move -->
      <div class="detail-status-bar">
        {#each columns as col}
          <button
            class="detail-status-btn"
            class:active={detailTicket.status === col.key}
            style="--col-color: {col.color}"
            on:click={() => { moveTicket(detailTicket.id, col.key); detailTicket.status = col.key }}
          >
            {col.label}
          </button>
        {/each}
      </div>

      <div class="detail-grid">
        <div class="detail-main">
          {#if detailTicket.description}
            <div class="detail-section">
              <h4>Description</h4>
              <p class="detail-text">{detailTicket.description}</p>
            </div>
          {/if}
          {#if detailTicket.scope}
            <div class="detail-section">
              <h4>Scope</h4>
              <p class="detail-text">{detailTicket.scope}</p>
            </div>
          {/if}
          {#if detailTicket.acceptanceCriteria && detailTicket.acceptanceCriteria.length > 0}
            <div class="detail-section">
              <h4>Acceptance Criteria</h4>
              <ul class="detail-ac">
                {#each detailTicket.acceptanceCriteria as ac}
                  <li>{ac}</li>
                {/each}
              </ul>
            </div>
          {/if}
          {#if detailTicket.technicalNotes}
            <div class="detail-section">
              <h4>Technical Notes</h4>
              <p class="detail-text mono">{detailTicket.technicalNotes}</p>
            </div>
          {/if}
        </div>
        <div class="detail-sidebar">
          <div class="detail-field"><span class="detail-label">Type</span><span>{detailTicket.type}</span></div>
          <div class="detail-field"><span class="detail-label">Priority</span><span style="color: {priorityColor(detailTicket.priority)}">{detailTicket.priority}</span></div>
          <div class="detail-field"><span class="detail-label">Assigned</span><span>{detailTicket.assignedTo ? agentName(detailTicket.assignedTo) : 'Unassigned'}</span></div>
          {#if epicName(detailTicket.epicId)}
            <div class="detail-field"><span class="detail-label">Epic</span><span>{epicName(detailTicket.epicId)}</span></div>
          {/if}
          {#if detailTicket.estimate}
            <div class="detail-field"><span class="detail-label">Estimate</span><span>{detailTicket.estimate}</span></div>
          {/if}
          {#if detailTicket.storyPoints}
            <div class="detail-field"><span class="detail-label">Points</span><span>{detailTicket.storyPoints}</span></div>
          {/if}
          {#if detailTicket.tags && detailTicket.tags.length > 0}
            <div class="detail-field"><span class="detail-label">Tags</span>
              <div class="tb-card-tags">{#each detailTicket.tags as tag}<span class="tb-tag">{tag}</span>{/each}</div>
            </div>
          {/if}
          {#if detailTicket.createdAt}
            <div class="detail-field"><span class="detail-label">Created</span><span class="detail-date">{detailTicket.createdAt.split('T')[0]}</span></div>
          {/if}
          {#if detailTicket.startedAt}
            <div class="detail-field"><span class="detail-label">Started</span><span class="detail-date">{detailTicket.startedAt.split('T')[0]}</span></div>
          {/if}
          {#if detailTicket.completedAt}
            <div class="detail-field"><span class="detail-label">Completed</span><span class="detail-date">{detailTicket.completedAt.split('T')[0]}</span></div>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Ticket Create/Edit Modal -->
{#if ticketModal}
  <div class="modal-overlay" on:click={() => ticketModal = false}>
    <div class="modal modal-lg" on:click|stopPropagation>
      <h2>{ticketMode === 'create' ? 'New Ticket' : 'Edit Ticket'}</h2>
      <div class="form-row">
        <div class="form-group flex-2">
          <label>Title *</label>
          <input type="text" bind:value={editingTicket.title} placeholder="Ticket title" />
        </div>
        <div class="form-group">
          <label>Type</label>
          <select bind:value={editingTicket.type}>
            <option value="feature">Feature</option>
            <option value="bug">Bug</option>
            <option value="chore">Chore</option>
            <option value="spike">Spike</option>
          </select>
        </div>
      </div>
      <div class="form-group">
        <label>Description</label>
        <textarea bind:value={editingTicket.description} placeholder="What needs to be done..." rows="3"></textarea>
      </div>
      <div class="form-group">
        <label>Scope</label>
        <textarea bind:value={editingTicket.scope} placeholder="Define the scope and boundaries..." rows="2"></textarea>
      </div>
      <div class="form-group">
        <label>Acceptance Criteria</label>
        <div class="ac-list">
          {#each editingTicket.acceptanceCriteria as ac, i}
            <div class="ac-item">
              <span class="ac-bullet">-</span>
              <span class="ac-text">{ac}</span>
              <button class="btn-icon-sm" on:click={() => removeAC(i)}>x</button>
            </div>
          {/each}
        </div>
        <div class="ac-input-row">
          <input type="text" bind:value={acInput} placeholder="Add criterion..." on:keydown={handleACKey} />
          <button class="tb-btn-secondary btn-sm" on:click={addAC}>Add</button>
        </div>
      </div>
      <div class="form-group">
        <label>Technical Notes</label>
        <textarea bind:value={editingTicket.technicalNotes} placeholder="Implementation details, architecture notes..." rows="2"></textarea>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Status</label>
          <select bind:value={editingTicket.status}>
            <option value="draft">Draft</option>
            <option value="ready">Ready</option>
            <option value="in_progress">In Progress</option>
            <option value="review">Review</option>
            <option value="done">Done</option>
          </select>
        </div>
        <div class="form-group">
          <label>Priority</label>
          <select bind:value={editingTicket.priority}>
            <option value="critical">Critical</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
          </select>
        </div>
        <div class="form-group">
          <label>Estimate</label>
          <select bind:value={editingTicket.estimate}>
            <option value="">None</option>
            <option value="S">S</option>
            <option value="M">M</option>
            <option value="L">L</option>
            <option value="XL">XL</option>
          </select>
        </div>
        <div class="form-group">
          <label>Story Points</label>
          <input type="number" bind:value={editingTicket.storyPoints} min="0" max="100" />
        </div>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Assigned To</label>
          <select bind:value={editingTicket.assignedTo}>
            <option value="">Unassigned</option>
            {#each projectAgents as agent}
              <option value={agent.agentId}>{agent.name} ({agent.agentId})</option>
            {/each}
          </select>
        </div>
        <div class="form-group">
          <label>Epic</label>
          <select bind:value={editingTicket.epicId}>
            <option value="">No Epic</option>
            {#each epics as ep}
              <option value={ep.id}>{ep.code} {ep.title}</option>
            {/each}
          </select>
        </div>
      </div>
      <div class="form-group">
        <label>Tags</label>
        <div class="tags-display">
          {#each editingTicket.tags as tag}
            <span class="tb-tag">{tag} <button class="tag-remove" on:click={() => removeTag(tag)}>x</button></span>
          {/each}
        </div>
        <input type="text" bind:value={tagInput} placeholder="Add tag (Enter)" on:keydown={handleTagKey} />
      </div>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => ticketModal = false}>Cancel</button>
        <button class="btn-primary" on:click={saveTicket} disabled={!editingTicket.title}>
          {ticketMode === 'create' ? 'Create Ticket' : 'Save Changes'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Epic Create/Edit Modal -->
{#if epicModal}
  <div class="modal-overlay" on:click={() => epicModal = false}>
    <div class="modal" on:click|stopPropagation>
      <h2>{epicMode === 'create' ? 'New Epic' : 'Edit Epic'}</h2>
      <div class="form-group">
        <label>Title *</label>
        <input type="text" bind:value={editingEpic.title} placeholder="Epic title" />
      </div>
      <div class="form-group">
        <label>Description</label>
        <textarea bind:value={editingEpic.description} placeholder="Epic description..." rows="3"></textarea>
      </div>
      <div class="form-group">
        <label>Status</label>
        <select bind:value={editingEpic.status}>
          <option value="open">Open</option>
          <option value="closed">Closed</option>
        </select>
      </div>
      <div class="modal-actions">
        {#if epicMode === 'edit'}
          <button class="tb-btn-danger" on:click={() => { deleteEpic(editingEpic.id); epicModal = false }}>Delete Epic</button>
        {/if}
        <div style="flex:1"></div>
        <button class="btn-secondary" on:click={() => epicModal = false}>Cancel</button>
        <button class="btn-primary" on:click={saveEpic} disabled={!editingEpic.title}>
          {epicMode === 'create' ? 'Create Epic' : 'Save'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Delete Confirm -->
{#if deleteConfirm}
  <div class="modal-overlay" on:click={() => deleteConfirm = null}>
    <div class="modal modal-small" on:click|stopPropagation>
      <h2>Delete Ticket</h2>
      <p>Delete <strong>{deleteConfirm.code} - {deleteConfirm.title}</strong>? This cannot be undone.</p>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => deleteConfirm = null}>Cancel</button>
        <button class="tb-btn-danger" on:click={deleteTicketConfirmed}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<!-- Generate Tickets Modal -->
{#if generateModal}
  <div class="modal-overlay" on:click={() => { if (!generating) generateModal = false }}>
    <div class="modal" on:click|stopPropagation>
      <h2>&#10024; Generate Tickets with AI</h2>
      <div class="form-group">
        <label>Describe the feature or task</label>
        <textarea
          bind:value={generateDescription}
          placeholder="Describe the feature you want to build..."
          rows="5"
          disabled={generating}
        ></textarea>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Create as Epic</label>
          <label class="gen-toggle">
            <input type="checkbox" bind:checked={generateCreateEpic} disabled={generating} />
            <span class="gen-toggle-label">{generateCreateEpic ? 'Yes' : 'No'}</span>
          </label>
        </div>
        <div class="form-group">
          <label>Initial Status</label>
          <div class="gen-radio-group">
            <label class="gen-radio">
              <input type="radio" bind:group={generateStatus} value="draft" disabled={generating} />
              Draft
            </label>
            <label class="gen-radio">
              <input type="radio" bind:group={generateStatus} value="ready" disabled={generating} />
              Ready to Work
            </label>
          </div>
        </div>
      </div>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => generateModal = false} disabled={generating}>Cancel</button>
        <button class="btn-primary" on:click={generateTickets} disabled={!generateDescription.trim() || generating}>
          {#if generating}
            <span class="gen-spinner"></span> Generating...
          {:else}
            Generate
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Notification Toast -->
{#if generateNotification}
  <div class="gen-toast" class:error={generateNotification.startsWith('Error') || generateNotification.startsWith('Generation failed')}>
    {generateNotification}
  </div>
{/if}

<style>
  .ticket-board {
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .tb-loading {
    text-align: center;
    color: var(--text-secondary);
    padding: 48px 0;
  }

  /* Stats */
  .tb-stats {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 0;
    font-size: 12px;
    color: var(--text-secondary);
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
  }
  .tb-stat { display: flex; align-items: center; gap: 4px; }
  .tb-stat-num { font-weight: 700; font-size: 14px; color: var(--text-primary); }
  .tb-progress { flex: 1; max-width: 120px; height: 4px; background: var(--bg-card); border-radius: 2px; overflow: hidden; }
  .tb-progress-fill { height: 100%; background: #22c55e; border-radius: 2px; transition: width 0.3s; }

  /* Toolbar */
  .tb-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    padding: 10px 0;
    flex-shrink: 0;
    flex-wrap: wrap;
  }
  .tb-toolbar-left, .tb-toolbar-right { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }

  .tb-btn-primary {
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
  .tb-btn-primary:hover { opacity: 0.85; }

  .tb-btn-secondary {
    background: var(--bg-card);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: var(--radius-sm);
    padding: 5px 12px;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }
  .tb-btn-secondary:hover { border-color: var(--accent); color: var(--accent); }
  .tb-btn-secondary.active { border-color: var(--accent); color: var(--accent); background: var(--accent)10; }

  .tb-btn-approve {
    background: #22c55e20;
    border: 1px solid #22c55e;
    color: #22c55e;
    border-radius: var(--radius-sm);
    padding: 5px 12px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }
  .tb-btn-approve:hover { background: #22c55e30; }

  .tb-btn-danger {
    background: #ef444420;
    border: 1px solid #ef4444;
    color: #ef4444;
    border-radius: var(--radius-sm);
    padding: 5px 12px;
    font-size: 12px;
    cursor: pointer;
  }
  .tb-btn-danger:hover { background: #ef444430; }

  .tb-btn-clear {
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: 11px;
    cursor: pointer;
    text-decoration: underline;
  }

  .tb-filter {
    background: var(--bg-card);
    border: 1px solid var(--border);
    color: var(--text-primary);
    border-radius: var(--radius-sm);
    padding: 5px 8px;
    font-size: 11px;
    cursor: pointer;
  }

  /* Epics bar */
  .tb-epics-bar {
    display: flex;
    gap: 8px;
    padding: 8px 0;
    flex-shrink: 0;
    overflow-x: auto;
  }
  .tb-epic-chip {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 4px 10px;
    font-size: 11px;
    cursor: pointer;
    white-space: nowrap;
    transition: border-color 0.15s;
  }
  .tb-epic-chip:hover { border-color: var(--accent); }
  .tb-epic-chip.closed { opacity: 0.5; }
  .tb-epic-code { font-weight: 700; color: var(--text-primary); }
  .tb-epic-title { color: var(--text-secondary); }
  .tb-epic-status { font-size: 10px; font-weight: 600; }

  /* Kanban */
  .tb-kanban {
    display: flex;
    gap: 12px;
    flex: 1;
    overflow-x: auto;
    overflow-y: hidden;
    padding-bottom: 8px;
  }

  .tb-column {
    flex: 1;
    min-width: 220px;
    max-width: 300px;
    display: flex;
    flex-direction: column;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius);
    overflow: hidden;
    transition: border-color 0.15s;
  }
  .tb-column.drag-over {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }

  .tb-col-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    border-bottom: 1px solid var(--border-subtle);
  }
  .tb-col-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-primary);
  }
  .tb-col-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .tb-col-count {
    font-size: 10px;
    color: var(--text-muted);
    background: var(--bg-base);
    padding: 1px 6px;
    border-radius: 10px;
    font-weight: 500;
  }
  .tb-btn-add {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    width: 22px;
    height: 22px;
    border-radius: 4px;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.15s;
  }
  .tb-btn-add:hover { border-color: var(--accent); color: var(--accent); }

  .tb-col-body {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  /* Cards */
  .tb-card {
    background: var(--bg-base);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: 10px;
    cursor: pointer;
    transition: all 0.15s;
    user-select: none;
  }
  .tb-card:hover { border-color: var(--border); box-shadow: 0 2px 8px rgba(0,0,0,0.15); }
  .tb-card.selected { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent); }
  .tb-card.dragging { opacity: 0.4; }

  .tb-card-check {
    margin-bottom: 6px;
  }
  .tb-card-check input { cursor: pointer; }

  .tb-card-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 4px;
  }
  .tb-card-code {
    font-size: 10px;
    font-weight: 700;
    color: var(--text-muted);
    font-family: var(--font-mono);
    letter-spacing: 0.3px;
  }
  .tb-card-badges { display: flex; gap: 4px; align-items: center; }

  .tb-badge-priority {
    font-size: 9px;
    font-weight: 700;
    padding: 1px 5px;
    border-radius: 3px;
    letter-spacing: 0.5px;
  }
  .tb-badge-estimate {
    font-size: 9px;
    font-weight: 600;
    padding: 1px 5px;
    border-radius: 3px;
    background: var(--bg-card);
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .tb-card-title {
    font-size: 12px;
    font-weight: 500;
    color: var(--text-primary);
    line-height: 1.3;
    margin-bottom: 4px;
    display: flex;
    align-items: flex-start;
    gap: 4px;
  }
  .tb-type-icon {
    color: var(--text-muted);
    font-size: 11px;
    flex-shrink: 0;
    margin-top: 1px;
  }

  .tb-card-epic {
    font-size: 10px;
    color: var(--accent);
    margin-bottom: 4px;
    opacity: 0.8;
  }

  .tb-card-tags { display: flex; flex-wrap: wrap; gap: 3px; margin-bottom: 4px; }
  .tb-tag {
    font-size: 9px;
    padding: 1px 5px;
    border-radius: 3px;
    background: var(--bg-card);
    color: var(--text-muted);
    border: 1px solid var(--border-subtle);
  }
  .tb-tag-more { font-style: italic; }
  .tag-remove {
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 0 2px;
    font-size: 10px;
  }

  .tb-card-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 10px;
  }
  .tb-assignee { color: var(--text-secondary); }
  .tb-unassigned { color: var(--text-muted); font-style: italic; }
  .tb-points {
    color: var(--text-muted);
    font-weight: 600;
    font-size: 10px;
  }

  /* Modals — reuse existing app modal styles */
  .modal-overlay {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(0,0,0,0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(2px);
  }
  .modal {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 24px;
    width: 480px;
    max-height: 80vh;
    overflow-y: auto;
  }
  .modal.modal-lg { width: 680px; }
  .modal.modal-small { width: 380px; }
  .modal h2 {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 16px;
  }

  .form-group { margin-bottom: 12px; }
  .form-group label {
    display: block;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 4px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .form-group input, .form-group textarea, .form-group select {
    width: 100%;
    background: var(--bg-base);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 8px 10px;
    color: var(--text-primary);
    font-size: 13px;
    font-family: inherit;
  }
  .form-group input:focus, .form-group textarea:focus, .form-group select:focus {
    outline: none;
    border-color: var(--accent);
  }
  .form-group textarea { resize: vertical; }

  .form-row {
    display: flex;
    gap: 12px;
  }
  .form-row .form-group { flex: 1; }
  .form-row .flex-2 { flex: 2; }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 16px;
    padding-top: 12px;
    border-top: 1px solid var(--border-subtle);
  }
  .btn-secondary {
    background: var(--bg-base);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: var(--radius-sm);
    padding: 6px 14px;
    font-size: 12px;
    cursor: pointer;
  }
  .btn-secondary:hover { border-color: var(--text-secondary); }
  .btn-primary {
    background: var(--accent);
    border: none;
    color: #fff;
    border-radius: var(--radius-sm);
    padding: 6px 14px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }
  .btn-primary:hover { opacity: 0.85; }
  .btn-primary:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-sm { padding: 4px 10px; font-size: 11px; }

  /* Acceptance criteria */
  .ac-list { margin-bottom: 6px; }
  .ac-item {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--text-secondary);
    padding: 3px 0;
  }
  .ac-bullet { color: var(--text-muted); }
  .ac-text { flex: 1; }
  .btn-icon-sm {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 11px;
    padding: 2px 4px;
  }
  .btn-icon-sm:hover { color: #ef4444; }
  .ac-input-row { display: flex; gap: 6px; }
  .ac-input-row input { flex: 1; }

  .tags-display { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 6px; }

  /* Detail modal */
  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  .detail-header-left { display: flex; align-items: center; gap: 8px; }
  .detail-header-right { display: flex; gap: 6px; }
  .detail-code {
    font-size: 13px;
    font-weight: 700;
    color: var(--text-muted);
    font-family: var(--font-mono);
  }
  .detail-title {
    font-size: 18px;
    margin-bottom: 12px;
  }

  .detail-status-bar {
    display: flex;
    gap: 4px;
    margin-bottom: 16px;
  }
  .detail-status-btn {
    flex: 1;
    padding: 6px 8px;
    font-size: 11px;
    font-weight: 600;
    background: var(--bg-base);
    border: 1px solid var(--border-subtle);
    color: var(--text-muted);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all 0.15s;
  }
  .detail-status-btn:hover { border-color: var(--col-color); color: var(--col-color); }
  .detail-status-btn.active {
    background: color-mix(in srgb, var(--col-color) 15%, transparent);
    border-color: var(--col-color);
    color: var(--col-color);
  }

  .detail-grid {
    display: flex;
    gap: 20px;
  }
  .detail-main { flex: 2; }
  .detail-sidebar {
    flex: 1;
    border-left: 1px solid var(--border-subtle);
    padding-left: 16px;
  }

  .detail-section { margin-bottom: 14px; }
  .detail-section h4 {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 6px;
  }
  .detail-text {
    font-size: 13px;
    color: var(--text-secondary);
    line-height: 1.5;
    white-space: pre-wrap;
  }
  .detail-text.mono { font-family: var(--font-mono); font-size: 12px; }
  .detail-ac {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  .detail-ac li {
    font-size: 12px;
    color: var(--text-secondary);
    padding: 3px 0;
    padding-left: 14px;
    position: relative;
  }
  .detail-ac li::before {
    content: '•';
    position: absolute;
    left: 0;
    color: var(--accent);
  }

  .detail-field {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    font-size: 12px;
    padding: 6px 0;
    border-bottom: 1px solid var(--border-subtle);
  }
  .detail-label {
    color: var(--text-muted);
    font-weight: 500;
  }
  .detail-date {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--text-secondary);
  }

  /* Generate button */
  .tb-btn-generate {
    background: linear-gradient(135deg, #8b5cf6, #6366f1);
    color: #fff;
    border: none;
    border-radius: var(--radius-sm);
    padding: 6px 14px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
  }
  .tb-btn-generate:hover { opacity: 0.85; }

  /* Generate modal extras */
  .gen-toggle {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 13px;
    color: var(--text-primary);
  }
  .gen-toggle input { cursor: pointer; }
  .gen-toggle-label { font-weight: 500; }

  .gen-radio-group {
    display: flex;
    gap: 16px;
  }
  .gen-radio {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 13px;
    color: var(--text-primary);
    cursor: pointer;
  }
  .gen-radio input { cursor: pointer; }

  .gen-spinner {
    display: inline-block;
    width: 12px;
    height: 12px;
    border: 2px solid rgba(255,255,255,0.3);
    border-top-color: #fff;
    border-radius: 50%;
    animation: gen-spin 0.6s linear infinite;
    vertical-align: middle;
    margin-right: 4px;
  }
  @keyframes gen-spin {
    to { transform: rotate(360deg); }
  }

  /* Toast notification */
  .gen-toast {
    position: fixed;
    bottom: 24px;
    right: 24px;
    background: #22c55e;
    color: #fff;
    padding: 10px 20px;
    border-radius: var(--radius-sm);
    font-size: 13px;
    font-weight: 600;
    z-index: 2000;
    box-shadow: 0 4px 12px rgba(0,0,0,0.3);
    animation: gen-toast-in 0.3s ease;
  }
  .gen-toast.error {
    background: #ef4444;
  }
  @keyframes gen-toast-in {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
  }
</style>

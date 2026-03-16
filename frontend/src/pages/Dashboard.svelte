<script>
  import { onMount, onDestroy } from 'svelte'

  const wails = window['go']?.['main']?.['App']

  let recentActivity = []
  let searchQuery = ''
  let searchResults = []
  let searching = false
  let loadingActivity = true

  // Knowledge form
  let showAddKnowledge = false
  let knowledgeForm = { type: 'note', title: '', content: '', tags: '', projectId: '' }
  let savingKnowledge = false

  const actionIcons = {
    created: '+',
    updated: '~',
    deleted: '-',
  }

  const typeIcons = {
    decision: '!',
    lesson: '*',
    spec: '#',
    note: '.',
    research: '?',
  }

  function timeAgo(ts) {
    if (!ts) return ''
    const diff = Date.now() - new Date(ts).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 60) return mins + 'm ago'
    const hrs = Math.floor(mins / 60)
    if (hrs < 24) return hrs + 'h ago'
    const days = Math.floor(hrs / 24)
    return days + 'd ago'
  }

  async function loadRecentActivity() {
    try {
      loadingActivity = true
      recentActivity = await wails.GetRecentActivity(10) || []
    } catch (e) {
      recentActivity = []
    } finally {
      loadingActivity = false
    }
  }

  async function searchKnowledge() {
    if (!searchQuery.trim()) {
      searchResults = []
      return
    }
    try {
      searching = true
      searchResults = await wails.SearchKnowledge(searchQuery, 10) || []
    } catch (e) {
      searchResults = []
    } finally {
      searching = false
    }
  }

  async function addKnowledge() {
    if (!knowledgeForm.title.trim()) return
    try {
      savingKnowledge = true
      await wails.AddKnowledge(
        knowledgeForm.type,
        knowledgeForm.title,
        knowledgeForm.content,
        knowledgeForm.tags,
        knowledgeForm.projectId
      )
      knowledgeForm = { type: 'note', title: '', content: '', tags: '', projectId: '' }
      showAddKnowledge = false
    } catch (e) {
      console.error('Failed to add knowledge:', e)
    } finally {
      savingKnowledge = false
    }
  }

  function handleSearchKeydown(e) {
    if (e.key === 'Enter') searchKnowledge()
  }

  let unsubActivity = null

  onMount(() => {
    loadRecentActivity()
    // Start sync
    wails.StartSync?.()
    // Listen for new activity
    const runtime = window.runtime
    if (runtime) {
      runtime.EventsOn('hivemind:new-activity', () => {
        loadRecentActivity()
      })
      unsubActivity = () => runtime.EventsOff('hivemind:new-activity')
    }
  })

  onDestroy(() => {
    if (unsubActivity) unsubActivity()
  })
</script>

<div class="dashboard">
  <div class="dashboard-grid">
    <!-- Activity Feed -->
    <div class="card activity-card">
      <div class="card-header">
        <h2 class="card-title">Recent Activity</h2>
        <span class="card-badge">Live</span>
      </div>
      <div class="activity-list">
        {#if loadingActivity}
          <div class="activity-empty">Loading...</div>
        {:else if recentActivity.length === 0}
          <div class="activity-empty">No activity yet</div>
        {:else}
          {#each recentActivity as item}
            <div class="activity-item">
              <div class="activity-icon" class:created={item.action === 'created'} class:updated={item.action === 'updated'} class:deleted={item.action === 'deleted'}>
                {actionIcons[item.action] || '.'}
              </div>
              <div class="activity-content">
                <span class="activity-text">{item.summary}</span>
                <div class="activity-meta">
                  <span class="activity-instance">{item.instanceId}</span>
                  <span class="activity-time">{timeAgo(item.timestamp)}</span>
                </div>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Knowledge Search -->
    <div class="card knowledge-card">
      <div class="card-header">
        <h2 class="card-title">Knowledge Base</h2>
        <button class="btn-add" on:click={() => showAddKnowledge = !showAddKnowledge}>
          {showAddKnowledge ? '-' : '+'}
        </button>
      </div>

      {#if showAddKnowledge}
        <div class="knowledge-form">
          <div class="form-row-inline">
            <select bind:value={knowledgeForm.type} class="form-select">
              <option value="note">Note</option>
              <option value="decision">Decision</option>
              <option value="lesson">Lesson</option>
              <option value="spec">Spec</option>
              <option value="research">Research</option>
            </select>
            <input type="text" bind:value={knowledgeForm.title} placeholder="Title" class="form-input" />
          </div>
          <textarea bind:value={knowledgeForm.content} placeholder="Content..." rows="3" class="form-textarea"></textarea>
          <input type="text" bind:value={knowledgeForm.tags} placeholder="Tags (comma-separated)" class="form-input" />
          <div class="form-actions">
            <button class="btn-cancel" on:click={() => showAddKnowledge = false}>Cancel</button>
            <button class="btn-save" on:click={addKnowledge} disabled={!knowledgeForm.title.trim() || savingKnowledge}>
              {savingKnowledge ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      {/if}

      <div class="search-bar">
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Search knowledge..."
          on:keydown={handleSearchKeydown}
          class="search-input"
        />
        <button class="search-btn" on:click={searchKnowledge} disabled={searching}>
          {searching ? '...' : 'Go'}
        </button>
      </div>

      <div class="search-results">
        {#if searchResults.length > 0}
          {#each searchResults as result}
            <div class="result-item">
              <div class="result-header">
                <span class="result-type">{typeIcons[result.type] || '.'} {result.type}</span>
                <span class="result-score">{(result.score * 100).toFixed(1)}%</span>
              </div>
              <div class="result-title">{result.title}</div>
              <div class="result-content">{result.content}</div>
              {#if result.tags && result.tags.length > 0}
                <div class="result-tags">
                  {#each result.tags as tag}
                    <span class="result-tag">{tag}</span>
                  {/each}
                </div>
              {/if}
            </div>
          {/each}
        {:else if searchQuery && !searching}
          <div class="activity-empty">No results</div>
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .dashboard-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    padding: 20px;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .card-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .card-badge {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--success);
    background: rgba(34, 197, 94, 0.1);
    padding: 3px 8px;
    border-radius: var(--radius-sm);
  }

  .activity-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
    max-height: 460px;
    overflow-y: auto;
  }

  .activity-item {
    display: flex;
    align-items: flex-start;
    gap: 10px;
  }

  .activity-icon {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 700;
    flex-shrink: 0;
    margin-top: 2px;
    background: var(--accent-subtle);
    color: var(--accent);
  }

  .activity-icon.created {
    background: rgba(34, 197, 94, 0.15);
    color: #22c55e;
  }

  .activity-icon.updated {
    background: rgba(59, 130, 246, 0.15);
    color: #3b82f6;
  }

  .activity-icon.deleted {
    background: rgba(239, 68, 68, 0.15);
    color: #ef4444;
  }

  .activity-content {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .activity-text {
    font-size: 13px;
    color: var(--text-primary);
    line-height: 1.3;
  }

  .activity-meta {
    display: flex;
    gap: 8px;
    font-size: 11px;
  }

  .activity-instance {
    color: var(--accent);
  }

  .activity-time {
    color: var(--text-muted);
  }

  .activity-empty {
    font-size: 12px;
    color: var(--text-muted);
    text-align: center;
    padding: 16px 0;
  }

  /* Knowledge Search */
  .search-bar {
    display: flex;
    gap: 6px;
    margin-bottom: 12px;
  }

  .search-input {
    flex: 1;
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

  .search-input:focus {
    border-color: var(--accent);
  }

  .search-btn {
    background: var(--accent);
    color: white;
    border: none;
    border-radius: var(--radius-sm);
    padding: 8px 14px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity var(--transition-fast);
  }

  .search-btn:hover { opacity: 0.9; }
  .search-btn:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-add {
    background: var(--accent-subtle);
    color: var(--accent);
    border: 1px solid var(--accent);
    border-radius: var(--radius-sm);
    width: 26px;
    height: 26px;
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all var(--transition-fast);
  }

  .btn-add:hover {
    background: var(--accent);
    color: white;
  }

  .search-results {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 340px;
    overflow-y: auto;
  }

  .result-item {
    background: var(--bg-sidebar);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: 10px;
  }

  .result-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 4px;
  }

  .result-type {
    font-size: 10px;
    text-transform: uppercase;
    color: var(--accent);
    font-weight: 600;
    letter-spacing: 0.3px;
  }

  .result-score {
    font-size: 10px;
    color: var(--success);
    font-weight: 600;
  }

  .result-title {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .result-content {
    font-size: 12px;
    color: var(--text-secondary);
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
  }

  .result-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 6px;
  }

  .result-tag {
    font-size: 10px;
    background: var(--accent-subtle);
    color: var(--accent);
    padding: 1px 6px;
    border-radius: 3px;
  }

  /* Knowledge Form */
  .knowledge-form {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 12px;
    padding: 12px;
    background: var(--bg-sidebar);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
  }

  .form-row-inline {
    display: flex;
    gap: 8px;
  }

  .form-select {
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 6px 8px;
    font-size: 12px;
    font-family: inherit;
    outline: none;
  }

  .form-input {
    flex: 1;
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 6px 8px;
    font-size: 12px;
    font-family: inherit;
    outline: none;
  }

  .form-textarea {
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 6px 8px;
    font-size: 12px;
    font-family: inherit;
    outline: none;
    resize: vertical;
  }

  .form-input:focus, .form-textarea:focus, .form-select:focus {
    border-color: var(--accent);
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 6px;
  }

  .btn-cancel {
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    padding: 5px 12px;
    font-size: 11px;
    cursor: pointer;
    font-family: inherit;
  }

  .btn-save {
    background: var(--accent);
    color: white;
    border: none;
    border-radius: var(--radius-sm);
    padding: 5px 12px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
  }

  .btn-save:disabled { opacity: 0.5; cursor: not-allowed; }

  @media (max-width: 900px) {
    .dashboard-grid {
      grid-template-columns: 1fr;
    }
  }
</style>

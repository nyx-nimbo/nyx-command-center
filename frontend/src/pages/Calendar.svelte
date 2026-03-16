<script>
  import { onMount } from 'svelte'

  let todayEvents = []
  let upcomingEvents = []
  let loadingToday = true
  let loadingUpcoming = true
  let errorToday = ''
  let errorUpcoming = ''
  let showAddEvent = false
  let newTitle = ''
  let newDescription = ''
  let newStartDate = ''
  let newStartTime = ''
  let newEndDate = ''
  let newEndTime = ''
  let creating = false
  let createError = ''
  let createSuccess = false

  const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

  onMount(() => {
    loadToday()
    loadUpcoming()
  })

  async function loadToday() {
    loadingToday = true
    errorToday = ''
    try {
      const result = await window['go']['main']['App']['GetTodayEvents']()
      if (result.error) {
        errorToday = result.error
      } else {
        todayEvents = result.events || []
      }
    } catch (e) {
      errorToday = e.toString()
    }
    loadingToday = false
  }

  async function loadUpcoming() {
    loadingUpcoming = true
    errorUpcoming = ''
    try {
      const result = await window['go']['main']['App']['GetUpcomingEvents'](7)
      if (result.error) {
        errorUpcoming = result.error
      } else {
        upcomingEvents = result.events || []
      }
    } catch (e) {
      errorUpcoming = e.toString()
    }
    loadingUpcoming = false
  }

  function refresh() {
    loadToday()
    loadUpcoming()
  }

  function formatTime(isoStr) {
    if (!isoStr) return ''
    const d = new Date(isoStr)
    const h = d.getHours()
    const m = d.getMinutes().toString().padStart(2, '0')
    const ampm = h >= 12 ? 'PM' : 'AM'
    const h12 = h % 12 || 12
    return `${h12}:${m} ${ampm}`
  }

  function formatTimeShort(isoStr) {
    if (!isoStr) return ''
    const d = new Date(isoStr)
    return d.getHours().toString().padStart(2, '0') + ':' + d.getMinutes().toString().padStart(2, '0')
  }

  function formatDuration(startStr, endStr) {
    if (!startStr || !endStr) return ''
    const start = new Date(startStr)
    const end = new Date(endStr)
    const mins = Math.round((end - start) / 60000)
    if (mins < 60) return `${mins}m`
    const h = Math.floor(mins / 60)
    const m = mins % 60
    return m > 0 ? `${h}h ${m}m` : `${h}h`
  }

  function formatUpcomingDay(isoStr) {
    if (!isoStr) return ''
    const d = new Date(isoStr)
    const now = new Date()
    const tomorrow = new Date(now)
    tomorrow.setDate(tomorrow.getDate() + 1)

    if (d.toDateString() === tomorrow.toDateString()) return 'Tomorrow'

    return `${days[d.getDay()]}, ${months[d.getMonth()]} ${d.getDate()}`
  }

  function getCurrentDay() {
    const now = new Date()
    return `${days[now.getDay()]}, ${months[now.getMonth()]} ${now.getDate()}`
  }

  function openAddEvent() {
    const now = new Date()
    const y = now.getFullYear()
    const mo = (now.getMonth() + 1).toString().padStart(2, '0')
    const da = now.getDate().toString().padStart(2, '0')
    newStartDate = `${y}-${mo}-${da}`
    newEndDate = `${y}-${mo}-${da}`
    newStartTime = '09:00'
    newEndTime = '10:00'
    newTitle = ''
    newDescription = ''
    createError = ''
    createSuccess = false
    showAddEvent = true
  }

  function closeAddEvent() {
    showAddEvent = false
  }

  async function createEvent() {
    if (!newTitle || !newStartDate || !newStartTime || !newEndDate || !newEndTime) {
      createError = 'Title, start and end are required'
      return
    }
    creating = true
    createError = ''
    createSuccess = false
    try {
      const startISO = `${newStartDate}T${newStartTime}`
      const endISO = `${newEndDate}T${newEndTime}`
      const result = await window['go']['main']['App']['CreateEvent'](newTitle, newDescription, startISO, endISO)
      if (result.error) {
        createError = result.error
      } else {
        createSuccess = true
        setTimeout(() => {
          showAddEvent = false
          refresh()
        }, 1000)
      }
    } catch (e) {
      createError = e.toString()
    }
    creating = false
  }

  async function deleteEvent(id) {
    try {
      await window['go']['main']['App']['DeleteEvent'](id)
      refresh()
    } catch (e) {
      // silently fail
    }
  }
</script>

<div class="calendar-page">
  <div class="calendar-header">
    <div class="header-actions">
      <button class="btn btn-primary" on:click={openAddEvent}>Add Event</button>
      <button class="btn btn-ghost" on:click={refresh} disabled={loadingToday && loadingUpcoming}>
        {#if loadingToday && loadingUpcoming}
          <span class="spinner"></span>
        {:else}
          Refresh
        {/if}
      </button>
    </div>
  </div>

  {#if errorToday}
    <div class="error-banner">{errorToday}</div>
  {/if}

  <div class="calendar-grid">
    <div class="card today-card">
      <div class="card-header">
        <h2 class="card-title">Today — {getCurrentDay()}</h2>
        {#if !loadingToday}
          <span class="event-count">{todayEvents.length} event{todayEvents.length !== 1 ? 's' : ''}</span>
        {/if}
      </div>

      {#if loadingToday}
        <div class="loading-state">
          <span class="spinner"></span>
          <span>Loading events...</span>
        </div>
      {:else if todayEvents.length === 0}
        <div class="empty-state">No events today</div>
      {:else}
        <div class="timeline">
          {#each todayEvents as event}
            <div class="timeline-event">
              <div class="event-time">{formatTimeShort(event.startTime)}</div>
              <div class="event-bar {event.color}"></div>
              <div class="event-details">
                <span class="event-title">{event.title}</span>
                <span class="event-duration">
                  {#if event.allDay}
                    All day
                  {:else}
                    {formatTime(event.startTime)} - {formatTime(event.endTime)} · {formatDuration(event.startTime, event.endTime)}
                  {/if}
                </span>
                {#if event.location}
                  <span class="event-location">{event.location}</span>
                {/if}
              </div>
              <button class="btn-delete" on:click|stopPropagation={() => deleteEvent(event.id)} title="Delete event">&times;</button>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="card upcoming-card">
      <div class="card-header">
        <h2 class="card-title">Upcoming (7 days)</h2>
      </div>

      {#if loadingUpcoming}
        <div class="loading-state">
          <span class="spinner"></span>
          <span>Loading...</span>
        </div>
      {:else if upcomingEvents.length === 0}
        <div class="empty-state">No upcoming events</div>
      {:else}
        <div class="upcoming-list">
          {#each upcomingEvents as event}
            <div class="upcoming-item">
              <div class="upcoming-day">{formatUpcomingDay(event.startTime)}</div>
              <div class="upcoming-details">
                <span class="upcoming-title">{event.title}</span>
                <span class="upcoming-time">
                  {#if event.allDay}
                    All day
                  {:else}
                    {formatTime(event.startTime)} - {formatTime(event.endTime)}
                  {/if}
                </span>
                {#if event.location}
                  <span class="upcoming-location">{event.location}</span>
                {/if}
              </div>
              <button class="btn-delete" on:click|stopPropagation={() => deleteEvent(event.id)} title="Delete event">&times;</button>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  <!-- Add Event Modal -->
  {#if showAddEvent}
    <div class="modal-overlay" on:click={closeAddEvent}>
      <div class="modal-panel" on:click|stopPropagation>
        <div class="modal-header">
          <div class="modal-title">New Event</div>
          <button class="btn-close" on:click={closeAddEvent}>&times;</button>
        </div>
        <div class="modal-form">
          <input
            class="form-input"
            type="text"
            placeholder="Event title"
            bind:value={newTitle}
          />
          <textarea
            class="form-textarea"
            placeholder="Description (optional)"
            bind:value={newDescription}
            rows="3"
          ></textarea>
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Start</label>
              <div class="datetime-row">
                <input class="form-input" type="date" bind:value={newStartDate} />
                <input class="form-input time-input" type="time" bind:value={newStartTime} />
              </div>
            </div>
            <div class="form-group">
              <label class="form-label">End</label>
              <div class="datetime-row">
                <input class="form-input" type="date" bind:value={newEndDate} />
                <input class="form-input time-input" type="time" bind:value={newEndTime} />
              </div>
            </div>
          </div>
          {#if createError}
            <div class="form-error">{createError}</div>
          {/if}
          {#if createSuccess}
            <div class="form-success">Event created!</div>
          {/if}
          <div class="form-actions">
            <button class="btn btn-primary" on:click={createEvent} disabled={creating}>
              {#if creating}
                <span class="spinner"></span> Creating...
              {:else}
                Create Event
              {/if}
            </button>
            <button class="btn btn-ghost" on:click={closeAddEvent}>Cancel</button>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .calendar-page {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .calendar-header {
    display: flex;
    justify-content: flex-end;
    align-items: center;
  }

  .header-actions {
    display: flex;
    gap: 8px;
  }

  .btn {
    padding: 8px 16px;
    border-radius: var(--radius-md);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: all var(--transition-fast);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .btn-primary {
    background: var(--accent);
    color: white;
  }

  .btn-primary:hover {
    opacity: 0.9;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-ghost {
    background: transparent;
    color: var(--text-secondary);
    border: 1px solid var(--border-subtle);
  }

  .btn-ghost:hover {
    background: var(--bg-card-hover);
    color: var(--text-primary);
  }

  .btn-ghost:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .error-banner {
    padding: 12px 16px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: var(--radius-md);
    color: #ef4444;
    font-size: 13px;
  }

  .calendar-grid {
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
    margin-bottom: 20px;
  }

  .card-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .event-count {
    font-size: 12px;
    color: var(--text-muted);
  }

  .loading-state,
  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding: 40px 20px;
    color: var(--text-muted);
    font-size: 13px;
  }

  .spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid var(--border-subtle);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .timeline {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .timeline-event {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .event-time {
    font-size: 12px;
    color: var(--text-muted);
    width: 48px;
    flex-shrink: 0;
    font-variant-numeric: tabular-nums;
  }

  .event-bar {
    width: 3px;
    height: 42px;
    border-radius: 2px;
    flex-shrink: 0;
  }

  .event-bar.accent { background: var(--accent); }
  .event-bar.success { background: var(--success); }
  .event-bar.warning { background: var(--warning); }
  .event-bar.info { background: #3b82f6; }

  .event-details {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }

  .event-title {
    font-size: 13px;
    color: var(--text-primary);
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .event-duration {
    font-size: 11px;
    color: var(--text-muted);
  }

  .event-location {
    font-size: 11px;
    color: var(--text-muted);
    font-style: italic;
  }

  .btn-delete {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 18px;
    cursor: pointer;
    padding: 2px 6px;
    opacity: 0;
    transition: opacity var(--transition-fast);
    flex-shrink: 0;
  }

  .timeline-event:hover .btn-delete,
  .upcoming-item:hover .btn-delete {
    opacity: 1;
  }

  .btn-delete:hover {
    color: #ef4444;
  }

  .upcoming-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .upcoming-item {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 10px 0;
    border-bottom: 1px solid var(--border-subtle);
  }

  .upcoming-item:last-child {
    border-bottom: none;
  }

  .upcoming-day {
    font-size: 12px;
    color: var(--accent);
    font-weight: 600;
    width: 100px;
    flex-shrink: 0;
  }

  .upcoming-details {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }

  .upcoming-title {
    font-size: 13px;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .upcoming-time {
    font-size: 11px;
    color: var(--text-muted);
  }

  .upcoming-location {
    font-size: 11px;
    color: var(--text-muted);
    font-style: italic;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 100;
    padding: 40px;
  }

  .modal-panel {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 520px;
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .modal-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .btn-close {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 24px;
    cursor: pointer;
    padding: 0;
    line-height: 1;
  }

  .btn-close:hover {
    color: var(--text-primary);
  }

  .modal-form {
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .form-input {
    padding: 10px 14px;
    background: var(--bg-main);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    font-size: 13px;
    outline: none;
    width: 100%;
    box-sizing: border-box;
  }

  .form-input:focus {
    border-color: var(--accent);
  }

  .time-input {
    width: 120px;
    flex-shrink: 0;
  }

  .form-textarea {
    padding: 10px 14px;
    background: var(--bg-main);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    font-size: 13px;
    font-family: inherit;
    resize: vertical;
    outline: none;
    width: 100%;
    box-sizing: border-box;
  }

  .form-textarea:focus {
    border-color: var(--accent);
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .form-label {
    font-size: 12px;
    color: var(--text-muted);
    font-weight: 500;
  }

  .datetime-row {
    display: flex;
    gap: 8px;
  }

  .form-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    margin-top: 4px;
  }

  .form-error {
    font-size: 12px;
    color: #ef4444;
  }

  .form-success {
    font-size: 12px;
    color: var(--success);
  }

  @media (max-width: 900px) {
    .calendar-grid {
      grid-template-columns: 1fr;
    }
    .form-row {
      grid-template-columns: 1fr;
    }
  }
</style>

<script lang="ts">
  import { dndzone } from 'svelte-dnd-action';
  import { apiFetch, app, fmtDateTimeWIB, notify, uploadFiles } from '../lib/state.svelte';
  import RutinitasBanner from '../components/RutinitasBanner.svelte';

  interface TaskItem {
    id: string;
    taskId: string;
    tutorId: string;
    title: string;
    description: string;
    category: string;
    priority: 'high' | 'medium' | 'low';
    metricCode?: string;
    dueAt: string;
    completedAt?: string;
    status: 'assigned' | 'overdue' | 'escalated' | 'completed' | 'cancelled';
    escalationReason?: string;
    cancelReason?: string;
    relatedStudentName?: string;
    relatedGroupTitle?: string;
    threadId?: string;
  }

  let tasks = $state<TaskItem[]>(app.cache.tasks || []);
  let search = $state('');
  let filterTile = $state<'all' | 'active' | 'overdue' | 'today' | 'escalated'>('all');
  let typeFilter = $state<'all' | 'kpi' | 'personal'>('all');
  let sortOption = $state<'sla_asc' | 'sla_desc' | 'priority'>('sla_asc');
  let compact = $state(false);
  let selectedIds = $state<string[]>([]);
  let activeTask = $state<TaskItem | null>(null);
  let linkedThread = $state<any>(null);

  // Per-column drag-and-drop state
  let columnsData = $state<Record<string, TaskItem[]>>({
    assigned: [],
    overdue: [],
    escalated: [],
    completed: [],
    cancelled: [],
  });

  // New Personal Task Modal
  let createOpen = $state(false);
  let newTitle = $state('');
  let newDesc = $state('');
  let newDue = $state('');
  let newPriority = $state<'low' | 'medium' | 'high'>('medium');

  // Escalate / Cancel Reason Modal
  let reasonModal = $state<{
    action: 'escalate' | 'cancel';
    ids: string[];
    revertFrom?: string;
  } | null>(null);
  let reasonInput = $state('');

  // Urgent Substitute Drawer
  let urgentOpen = $state(false);
  let eligibleLessons = $state<any[]>([]);
  let urgentRequests = $state<any[]>([]);
  let urgLessonId = $state('');
  let urgReason = $state('');
  let urgProofUrls = $state<string[]>([]);
  let urgUploading = $state(false);

  const columns = [
    { key: 'assigned', label: 'Baru / Aktif', color: 'border-indigo-500/40 bg-indigo-500/10 text-indigo-300' },
    { key: 'overdue', label: 'Terlambat', color: 'border-rose-500/40 bg-rose-500/10 text-rose-300' },
    { key: 'escalated', label: 'Dieskalasi', color: 'border-amber-500/40 bg-amber-500/10 text-amber-300' },
    { key: 'completed', label: 'Selesai', color: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-300' },
    { key: 'cancelled', label: 'Dibatalkan', color: 'border-slate-700 bg-slate-800/60 text-slate-400' },
  ];

  function syncColumns(rawTasks: TaskItem[]) {
    const map: Record<string, TaskItem[]> = {
      assigned: [],
      overdue: [],
      escalated: [],
      completed: [],
      cancelled: [],
    };
    for (const t of rawTasks) {
      const item = { ...t, id: t.taskId };
      if (map[item.status]) {
        map[item.status].push(item);
      } else {
        map['assigned'].push(item);
      }
    }
    columnsData = map;
  }

  async function loadTasks() {
    try {
      const res = await apiFetch<{ tasks: TaskItem[] }>('/api/v1/tasks');
      const loaded = (res.tasks || []).map((t) => ({ ...t, id: t.taskId }));
      tasks = loaded;
      app.cache.tasks = loaded;
      syncColumns(loaded);
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  async function moveTaskDirect(taskId: string, targetStatus: string, reason = '') {
    try {
      const res = await apiFetch(`/api/v1/tasks/${taskId}/move`, {
        method: 'PATCH',
        body: JSON.stringify({ targetStatus, reason }),
      });
      if (res.tasks) {
        tasks = res.tasks.map((t: any) => ({ ...t, id: t.taskId }));
        app.cache.tasks = tasks;
        syncColumns(tasks);
      } else {
        await loadTasks();
      }
      notify(`Tugas dipindahkan ke status '${targetStatus}'!`, 'success');
    } catch (err: any) {
      notify(err.message, 'error');
      await loadTasks();
    }
  }

  async function runTaskAction(ids: string[], action: string, reason = '') {
    try {
      const res = await apiFetch('/api/v1/tasks/bulk', {
        method: 'PATCH',
        body: JSON.stringify({ taskIds: ids, action, reason }),
      });
      tasks = (res.tasks || []).map((t: any) => ({ ...t, id: t.taskId }));
      app.cache.tasks = tasks;
      syncColumns(tasks);
      selectedIds = [];
      reasonModal = null;
      reasonInput = '';
      if (activeTask) {
        activeTask = tasks.find((t) => t.taskId === activeTask!.taskId) || null;
      }
      notify(`Tugas berhasil diperbarui (${action})!`, 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  // Drag and Drop handlers
  function handleDndConsider(colKey: string, e: CustomEvent<{ items: TaskItem[] }>) {
    columnsData[colKey] = e.detail.items;
  }

  async function handleDndFinalize(colKey: string, e: CustomEvent<{ items: TaskItem[]; info: { id: string; trigger: string } }>) {
    columnsData[colKey] = e.detail.items;
    const movedId = e.detail.info.id;
    if (!movedId) return;

    // Check if this column is the recipient of the moved item
    const isRecipient = e.detail.items.some((item) => item.id === movedId);
    if (!isRecipient) return;

    const originalTask = tasks.find((t) => t.taskId === movedId);
    if (!originalTask || originalTask.status === colKey) {
      // Reordered within same column or not found
      return;
    }

    const previousStatus = originalTask.status;

    // Handle business rules by target column
    if (colKey === 'escalated') {
      reasonModal = {
        action: 'escalate',
        ids: [movedId],
        revertFrom: previousStatus,
      };
      return;
    }

    if (colKey === 'cancelled') {
      if (originalTask.metricCode) {
        notify('Tugas KPI wajib tidak dapat dibatalkan!', 'error');
        await loadTasks();
        return;
      }
      reasonModal = {
        action: 'cancel',
        ids: [movedId],
        revertFrom: previousStatus,
      };
      return;
    }

    // Direct status move for completed, assigned, overdue
    await moveTaskDirect(movedId, colKey);
  }

  function cancelReasonModal() {
    reasonModal = null;
    reasonInput = '';
    loadTasks();
  }

  async function confirmReasonModal() {
    if (!reasonModal) return;
    if (reasonInput.trim().length < 5) {
      notify('Alasan minimal 5 karakter!', 'error');
      return;
    }
    const targetStatus = reasonModal.action === 'escalate' ? 'escalated' : 'cancelled';
    if (reasonModal.ids.length === 1) {
      await moveTaskDirect(reasonModal.ids[0], targetStatus, reasonInput);
    } else {
      await runTaskAction(reasonModal.ids, reasonModal.action, reasonInput);
    }
    reasonModal = null;
    reasonInput = '';
  }

  async function submitNewTask(e: Event) {
    e.preventDefault();
    try {
      const dueIso = newDue ? new Date(newDue).toISOString() : '';
      await apiFetch('/api/v1/tasks', {
        method: 'POST',
        body: JSON.stringify({
          title: newTitle,
          description: newDesc,
          dueAt: dueIso,
          priority: newPriority,
        }),
      });
      createOpen = false;
      newTitle = '';
      newDesc = '';
      newDue = '';
      await loadTasks();
      notify('Tugas pribadi berhasil ditambahkan!', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  async function openTaskDetail(tk: TaskItem) {
    activeTask = tk;
    linkedThread = null;
    if (tk.threadId) {
      try {
        const res = await apiFetch(`/api/v1/threads/${tk.threadId}`);
        linkedThread = res.thread;
      } catch {
        // ignore
      }
    }
  }

  async function openUrgentDrawer() {
    urgentOpen = true;
    try {
      const res = await apiFetch('/api/v1/urgent-subs');
      eligibleLessons = res.eligibleLessons || [];
      urgentRequests = res.requests || [];
      if (eligibleLessons.length > 0 && !urgLessonId) {
        urgLessonId = eligibleLessons[0].lessonId;
      }
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  async function handleUrgentFiles(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files?.length) return;
    urgUploading = true;
    try {
      const urls = await uploadFiles(input.files, 'proof');
      urgProofUrls = [...urgProofUrls, ...urls].slice(0, 3);
      notify(`${urls.length} bukti gambar diunggah`, 'info');
    } catch (err: any) {
      notify(err.message, 'error');
    } finally {
      urgUploading = false;
    }
  }

  async function submitUrgentSub(e: Event) {
    e.preventDefault();
    const ls = eligibleLessons.find((l) => l.lessonId === urgLessonId);
    try {
      await apiFetch('/api/v1/urgent-subs', {
        method: 'POST',
        body: JSON.stringify({
          lmsLessonId: urgLessonId,
          courseTitle: ls?.groupTitle || 'Coding Class',
          reason: urgReason,
          proofUrls: urgProofUrls,
        }),
      });
      urgReason = '';
      urgProofUrls = [];
      await openUrgentDrawer();
      notify('Permintaan guru pengganti mendesak terkirim!', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  async function withdrawUrgentSub(id: string) {
    try {
      await apiFetch(`/api/v1/urgent-subs/${id}`, { method: 'DELETE' });
      await openUrgentDrawer();
      notify('Permintaan pengganti mendesak ditarik.', 'info');
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  function toggleSelect(id: string) {
    if (selectedIds.includes(id)) {
      selectedIds = selectedIds.filter((x) => x !== id);
    } else {
      selectedIds = [...selectedIds, id];
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (['INPUT', 'TEXTAREA', 'SELECT'].includes((e.target as HTMLElement)?.tagName)) return;
    const targets = selectedIds.length > 0 ? selectedIds : activeTask ? [activeTask.taskId] : [];
    if (targets.length === 0) return;
    if (e.key === 'c' || e.key === 'C') {
      e.preventDefault();
      runTaskAction(targets, 'complete');
    } else if (e.key === 'e' || e.key === 'E') {
      e.preventDefault();
      reasonModal = { action: 'escalate', ids: targets };
    } else if (e.key === 'x' || e.key === 'X') {
      e.preventDefault();
      reasonModal = { action: 'cancel', ids: targets };
    }
  }

  $effect(() => {
    loadTasks();
  });

  const stats = $derived.by(() => {
    const todayStr = new Date().toISOString().slice(0, 10);
    return {
      active: tasks.filter((t) => t.status === 'assigned').length,
      overdue: tasks.filter((t) => t.status === 'overdue').length,
      dueToday: tasks.filter((t) => (t.status === 'assigned' || t.status === 'overdue') && (t.dueAt || '').startsWith(todayStr)).length,
      escalated: tasks.filter((t) => t.status === 'escalated').length,
    };
  });

  function getDisplayList(colKey: string): TaskItem[] {
    const list = columnsData[colKey] || [];
    const q = search.trim().toLowerCase();
    const todayStr = new Date().toISOString().slice(0, 10);

    return list
      .filter((t) => {
        if (filterTile === 'active' && t.status !== 'assigned') return false;
        if (filterTile === 'overdue' && t.status !== 'overdue') return false;
        if (filterTile === 'today' && !(t.dueAt || '').startsWith(todayStr)) return false;
        if (filterTile === 'escalated' && t.status !== 'escalated') return false;
        if (typeFilter === 'kpi' && !t.metricCode) return false;
        if (typeFilter === 'personal' && t.metricCode) return false;
        if (q && !t.title.toLowerCase().includes(q) && !(t.relatedStudentName || '').toLowerCase().includes(q)) return false;
        return true;
      })
      .sort((a, b) => {
        if (sortOption === 'sla_asc') return (a.dueAt || '').localeCompare(b.dueAt || '');
        if (sortOption === 'sla_desc') return (b.dueAt || '').localeCompare(a.dueAt || '');
        if (sortOption === 'priority') {
          const pOrder: Record<string, number> = { high: 3, medium: 2, low: 1 };
          return (pOrder[b.priority] || 0) - (pOrder[a.priority] || 0);
        }
        return 0;
      });
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="space-y-4">
  <!-- Top Banner: Rutinitas Mingguan Operasional -->
  <RutinitasBanner />

  <!-- Action Bar & Header -->
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-bold text-white flex items-center gap-2">
        <span>📋 Papan Tugas Mingguan (Interactive DnD Board)</span>
      </h2>
      <p class="text-xs text-slate-400 mt-0.5">
        Tarik & letakkan (drag and drop) kartu antar-kolom untuk memperbarui status seketika.
        Pintasan: <kbd class="px-1.5 py-0.5 rounded bg-slate-800 text-slate-200">C</kbd> Selesai,
        <kbd class="px-1.5 py-0.5 rounded bg-slate-800 text-slate-200">E</kbd> Eskalasi,
        <kbd class="px-1.5 py-0.5 rounded bg-slate-800 text-slate-200">X</kbd> Batalkan.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={openUrgentDrawer}
        class="px-3.5 py-2 rounded-xl bg-rose-600/20 hover:bg-rose-600/30 border border-rose-500/40 text-rose-300 text-xs font-semibold cursor-pointer transition shadow-xs"
      >
        🚨 Pengganti Mendesak (Urgent Sub)
      </button>
      <button
        onclick={() => (createOpen = true)}
        class="px-3.5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer transition shadow-md shadow-indigo-600/30"
      >
        + Buat Tugas Pribadi
      </button>
    </div>
  </div>

  <!-- 4 KPI Filter Tiles -->
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
    {#each [
      ['active', 'Aktif', stats.active, 'text-indigo-400'],
      ['overdue', 'Terlambat', stats.overdue, 'text-rose-400'],
      ['today', 'Jatuh Tempo Hari Ini', stats.dueToday, 'text-amber-400'],
      ['escalated', 'Dieskalasi ke TL', stats.escalated, 'text-orange-400'],
    ] as [key, label, val, col]}
      <button
        onclick={() => (filterTile = filterTile === key ? 'all' : (key as any))}
        class="p-3.5 rounded-2xl text-left border transition cursor-pointer {filterTile === key ? 'bg-slate-800 border-indigo-500 shadow-md' : 'bg-slate-900 border-slate-800 hover:border-slate-700'}"
      >
        <div class="text-[11px] uppercase font-semibold text-slate-400">{label}</div>
        <div class="mt-1 text-2xl font-black {col}">{val}</div>
      </button>
    {/each}
  </div>

  <!-- Filter Comboboxes & Search Bar -->
  <div class="flex flex-wrap items-center justify-between gap-3 bg-slate-900 border border-slate-800 p-3 rounded-2xl">
    <div class="flex-1 min-w-56">
      <input
        bind:value={search}
        placeholder="Cari judul tugas atau nama siswa..."
        class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white focus:outline-none focus:border-indigo-500"
      />
    </div>

    <div class="flex flex-wrap items-center gap-2 text-xs">
      <!-- Type Filter -->
      <select
        bind:value={typeFilter}
        class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-300 focus:outline-none focus:border-indigo-500 cursor-pointer"
      >
        <option value="all">Semua Jenis Tugas</option>
        <option value="kpi">Tugas KPI Algonova</option>
        <option value="personal">Pengingat Pribadi</option>
      </select>

      <!-- Sort Filter -->
      <select
        bind:value={sortOption}
        class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-300 focus:outline-none focus:border-indigo-500 cursor-pointer"
      >
        <option value="sla_asc">SLA Terdekat (Prioritas Waktu)</option>
        <option value="sla_desc">SLA Terjauh</option>
        <option value="priority">Tingkat Prioritas (High → Low)</option>
      </select>

      <!-- Compact Mode Toggle -->
      <button
        onclick={() => (compact = !compact)}
        class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-300 hover:text-white cursor-pointer transition"
      >
        Mode: {compact ? 'Ringkas' : 'Standar'}
      </button>
    </div>
  </div>

  <!-- 5-Column Kanban Board with svelte-dnd-action -->
  <div class="grid grid-cols-1 md:grid-cols-3 xl:grid-cols-5 gap-3.5 items-start">
    {#each columns as col}
      {@const colList = getDisplayList(col.key)}
      <div class="bg-slate-900/70 border border-slate-800 rounded-2xl p-3 flex flex-col min-h-[460px]">
        <!-- Column Header -->
        <div class="flex items-center justify-between mb-3 pb-2 border-b border-slate-800">
          <span class="text-xs font-bold px-2.5 py-0.5 rounded-full border {col.color}">{col.label}</span>
          <span class="text-xs font-bold text-slate-400">{colList.length}</span>
        </div>

        <!-- DnD Drop Zone -->
        <div
          use:dndzone={{
            items: colList,
            flipDuration_ms: 180,
            dropTargetStyle: {
              outline: '2px dashed #6366f1',
              borderRadius: '0.75rem',
              backgroundColor: 'rgba(99, 102, 241, 0.08)',
            },
          }}
          onconsider={(e) => handleDndConsider(col.key, e)}
          onfinalize={(e) => handleDndFinalize(col.key, e)}
          class="space-y-2.5 flex-1 min-h-[360px]"
        >
          {#each colList as tk (tk.id)}
            <div
              class="rounded-xl bg-slate-950 border transition shadow-sm {selectedIds.includes(tk.taskId) ? 'border-indigo-500 ring-1 ring-indigo-500' : 'border-slate-800 hover:border-slate-700'} {compact ? 'p-2.5' : 'p-3.5'} cursor-grab active:cursor-grabbing"
            >
              <div class="flex items-start gap-2">
                <input
                  type="checkbox"
                  checked={selectedIds.includes(tk.taskId)}
                  onchange={() => toggleSelect(tk.taskId)}
                  class="mt-0.5 rounded bg-slate-900 border-slate-700 cursor-pointer"
                />
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="flex-1 min-w-0" onclick={() => openTaskDetail(tk)}>
                  <div class="flex items-center gap-1.5 mb-1">
                    <span class="text-[10px] font-bold uppercase px-1.5 py-0.5 rounded {tk.priority === 'high' ? 'bg-rose-500/20 text-rose-300' : tk.priority === 'medium' ? 'bg-amber-500/20 text-amber-300' : 'bg-slate-800 text-slate-400'}">
                      {tk.priority}
                    </span>
                    {#if tk.metricCode}
                      <span class="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-indigo-500/20 text-indigo-300">KPI</span>
                    {/if}
                  </div>
                  <div class="text-xs font-semibold text-white leading-snug">{tk.title}</div>
                  {#if !compact && tk.relatedStudentName}
                    <div class="mt-1 text-[11px] text-slate-400 truncate">👤 {tk.relatedStudentName}</div>
                  {/if}
                  {#if tk.dueAt}
                    <div class="mt-1.5 text-[10px] text-slate-500">⏰ {fmtDateTimeWIB(tk.dueAt)}</div>
                  {/if}
                </div>
              </div>

              {#if tk.status === 'assigned' || tk.status === 'overdue'}
                <div class="mt-2.5 pt-2 border-t border-slate-800/80 flex items-center justify-end gap-1.5 text-[11px]">
                  <button
                    onclick={() => runTaskAction([tk.taskId], 'complete')}
                    class="px-2 py-0.5 rounded bg-emerald-600/20 hover:bg-emerald-600/35 text-emerald-300 font-semibold cursor-pointer transition"
                  >
                    Selesai
                  </button>
                  <button
                    onclick={() => (reasonModal = { action: 'escalate', ids: [tk.taskId] })}
                    class="px-2 py-0.5 rounded bg-amber-600/20 hover:bg-amber-600/35 text-amber-300 cursor-pointer transition"
                  >
                    Eskalasi
                  </button>
                </div>
              {/if}
            </div>
          {/each}

          {#if colList.length === 0}
            <div class="h-32 rounded-xl border border-dashed border-slate-800/80 flex items-center justify-center text-[11px] text-slate-600">
              Tarik kartu ke sini
            </div>
          {/if}
        </div>
      </div>
    {/each}
  </div>
</div>

<!-- Floating Sticky Batch Action Bar (shown when selectedIds.length > 0) -->
{#if selectedIds.length > 0}
  <div class="fixed bottom-6 left-1/2 -translate-x-1/2 z-40 bg-slate-900/95 backdrop-blur-md border border-indigo-500/50 shadow-2xl shadow-indigo-950/90 px-5 py-3 rounded-2xl flex items-center gap-4 text-xs">
    <div class="flex items-center gap-2">
      <span class="w-2.5 h-2.5 rounded-full bg-indigo-500 animate-pulse"></span>
      <span class="font-bold text-white"><span class="text-indigo-400">{selectedIds.length}</span> tugas terpilih</span>
    </div>
    <div class="h-4 w-px bg-slate-700"></div>
    <div class="flex items-center gap-2">
      <button
        onclick={() => runTaskAction(selectedIds, 'complete')}
        class="px-3 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold cursor-pointer transition shadow-md shadow-emerald-600/30"
      >
        ✓ Selesaikan ({selectedIds.length})
      </button>
      <button
        onclick={() => (reasonModal = { action: 'escalate', ids: selectedIds })}
        class="px-3 py-1.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-white font-bold cursor-pointer transition shadow-md shadow-amber-600/30"
      >
        ↑ Eskalasi ({selectedIds.length})
      </button>
      <button
        onclick={() => (reasonModal = { action: 'cancel', ids: selectedIds })}
        class="px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 font-semibold cursor-pointer transition"
      >
        ✕ Batalkan
      </button>
      <button
        onclick={() => (selectedIds = [])}
        class="px-2.5 py-1.5 rounded-xl text-slate-400 hover:text-white cursor-pointer transition"
      >
        Batal Pilihan
      </button>
    </div>
  </div>
{/if}

<!-- Task Detail Drawer -->
{#if activeTask}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-40 bg-black/65 flex justify-end" onclick={(e) => e.target === e.currentTarget && (activeTask = null)}>
    <div class="w-full max-w-lg bg-slate-900 border-l border-slate-800 h-full overflow-y-auto p-6 space-y-4">
      <div class="flex items-start justify-between">
        <div>
          <span class="text-[11px] font-bold uppercase px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-300">{activeTask.status}</span>
          <h3 class="mt-2 text-lg font-bold text-white">{activeTask.title}</h3>
        </div>
        <button onclick={() => (activeTask = null)} class="px-2.5 py-1 rounded bg-slate-800 text-xs text-slate-300 cursor-pointer">✕</button>
      </div>

      <p class="text-xs text-slate-300 leading-relaxed bg-slate-950 p-3.5 rounded-xl border border-slate-800">
        {activeTask.description || 'Tidak ada catatan tambahan.'}
      </p>

      <div class="grid grid-cols-2 gap-2.5 text-xs">
        <div class="p-3 rounded-xl bg-slate-950 border border-slate-800">
          <div class="text-slate-500">Tenggat SLA</div>
          <div class="font-semibold text-white mt-0.5">{fmtDateTimeWIB(activeTask.dueAt)}</div>
        </div>
        <div class="p-3 rounded-xl bg-slate-950 border border-slate-800">
          <div class="text-slate-500">Siswa / Grup Terkait</div>
          <div class="font-semibold text-white mt-0.5">{activeTask.relatedStudentName || activeTask.relatedGroupTitle || '-'}</div>
        </div>
      </div>

      {#if linkedThread}
        <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
          <div class="text-xs font-bold text-indigo-300">💬 Percakapan Terkait: {linkedThread.subject}</div>
          {#each linkedThread.messages || [] as m}
            <div class="text-xs p-2 rounded-lg bg-slate-900">
              <span class="font-semibold text-slate-200">{m.senderName}:</span>
              <span class="text-slate-400"> {m.body}</span>
            </div>
          {/each}
        </div>
      {/if}

      <div class="flex flex-wrap gap-2 pt-2">
        <button onclick={() => runTaskAction([activeTask!.taskId], 'complete')} class="px-4 py-2 rounded-xl bg-emerald-600 text-white text-xs font-semibold cursor-pointer">
          ✓ Tandai Selesai
        </button>
        <button onclick={() => (reasonModal = { action: 'escalate', ids: [activeTask!.taskId] })} class="px-4 py-2 rounded-xl bg-amber-600 text-white text-xs font-semibold cursor-pointer">
          ↑ Eskalasi ke TL
        </button>
        {#if !activeTask.metricCode}
          <button onclick={() => (reasonModal = { action: 'cancel', ids: [activeTask!.taskId] })} class="px-4 py-2 rounded-xl bg-slate-800 text-slate-300 text-xs cursor-pointer">
            Batalkan Tugas
          </button>
        {:else}
          <span class="text-[11px] text-slate-500 self-center">Tugas KPI tidak dapat dibatalkan</span>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Escalate / Cancel Reason Modal -->
{#if reasonModal}
  <div class="fixed inset-0 z-50 bg-black/70 flex items-center justify-center p-4">
    <div class="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-4">
      <h3 class="text-base font-bold text-white">
        {reasonModal.action === 'escalate' ? 'Alasan Eskalasi ke Team Lead' : 'Alasan Pembatalan Tugas'}
      </h3>
      <textarea
        bind:value={reasonInput}
        rows="3"
        placeholder="Tuliskan alasan minimal 5 karakter..."
        class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"
      ></textarea>
      <div class="flex justify-end gap-2">
        <button onclick={cancelReasonModal} class="px-3.5 py-2 rounded-xl bg-slate-800 text-xs text-slate-300 cursor-pointer">Batal</button>
        <button
          onclick={confirmReasonModal}
          class="px-4 py-2 rounded-xl bg-indigo-600 text-white text-xs font-semibold cursor-pointer"
        >
          Konfirmasi
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Create Personal Task Modal -->
{#if createOpen}
  <div class="fixed inset-0 z-50 bg-black/70 flex items-center justify-center p-4">
    <form onsubmit={submitNewTask} class="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-4">
      <h3 class="text-base font-bold text-white">Buat Tugas Pengingat Baru</h3>
      <input
        bind:value={newTitle}
        required
        minlength="3"
        placeholder="Judul tugas (mis. Cek project siswa atau persiapan modul)"
        class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"
      />
      <textarea
        bind:value={newDesc}
        rows="3"
        placeholder="Deskripsi opsional..."
        class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"
      ></textarea>
      <div class="grid grid-cols-2 gap-3">
        <input type="datetime-local" bind:value={newDue} class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white" />
        <select bind:value={newPriority} class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white">
          <option value="low">Prioritas Rendah</option>
          <option value="medium">Prioritas Sedang</option>
          <option value="high">Prioritas Tinggi</option>
        </select>
      </div>
      <div class="flex justify-end gap-2">
        <button type="button" onclick={() => (createOpen = false)} class="px-3.5 py-2 rounded-xl bg-slate-800 text-xs text-slate-300 cursor-pointer">Batal</button>
        <button type="submit" class="px-4 py-2 rounded-xl bg-indigo-600 text-white text-xs font-semibold cursor-pointer">Simpan Tugas</button>
      </div>
    </form>
  </div>
{/if}

<!-- Urgent Substitute Drawer -->
{#if urgentOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 bg-black/70 flex justify-end" onclick={(e) => e.target === e.currentTarget && (urgentOpen = false)}>
    <div class="w-full max-w-lg bg-slate-900 border-l border-slate-800 h-full overflow-y-auto p-6 space-y-5">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-lg font-bold text-white">Pengganti Mendesak (Urgent Sub)</h3>
          <p class="text-xs text-slate-400">Untuk sesi kelas dalam jendela 50 menit s/d 72 jam ke depan.</p>
        </div>
        <button onclick={() => (urgentOpen = false)} class="px-2.5 py-1 rounded bg-slate-800 text-xs text-slate-300 cursor-pointer">✕</button>
      </div>

      <form onsubmit={submitUrgentSub} class="space-y-3.5 p-4 rounded-2xl bg-slate-950 border border-slate-800">
        <div>
          <label for="urg-ls" class="block text-xs font-semibold text-slate-300 mb-1">Pilih Sesi Kelas (50m – 72j)</label>
          <select id="urg-ls" bind:value={urgLessonId} required class="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-xs text-white">
            {#each eligibleLessons as ls}
              <option value={ls.lessonId}>{ls.groupTitle} — {fmtDateTimeWIB(ls.startsAt)}</option>
            {/each}
          </select>
        </div>

        <div>
          <label for="urg-rsn" class="block text-xs font-semibold text-slate-300 mb-1">Alasan Darurat</label>
          <textarea
            id="urg-rsn"
            bind:value={urgReason}
            required
            rows="3"
            placeholder="Jelaskan kendala mendesak dan poin materi khusus untuk guru pengganti..."
            class="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-xs text-white"
          ></textarea>
        </div>

        <div>
          <label for="urg-file" class="block text-xs font-semibold text-slate-300 mb-1">Unggah Bukti Pendukung (1–3 Gambar)</label>
          <input id="urg-file" type="file" accept="image/*" multiple onchange={handleUrgentFiles} class="text-xs text-slate-400" />
          {#if urgUploading}<div class="text-xs text-indigo-400 mt-1">Mengunggah...</div>{/if}
          {#if urgProofUrls.length > 0}
            <div class="mt-2 flex gap-2">
              {#each urgProofUrls as u}
                <span class="text-[11px] px-2 py-1 rounded bg-emerald-500/15 text-emerald-300">✓ Bukti terlampir</span>
              {/each}
            </div>
          {/if}
        </div>

        <button type="submit" class="w-full py-2.5 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold cursor-pointer transition">
          Ajukan Guru Pengganti Sekarang
        </button>
      </form>

      {#if urgentRequests.length > 0}
        <div class="space-y-2">
          <h4 class="text-xs font-bold uppercase text-slate-400">Riwayat Pengajuan Urgent Sub</h4>
          {#each urgentRequests as r}
            <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between text-xs">
              <div>
                <div class="font-semibold text-white">{r.groupTitle}</div>
                <div class="text-slate-400">{fmtDateTimeWIB(r.lessonStartsAt)} • Status: <span class="text-amber-300 font-semibold">{r.status}</span></div>
              </div>
              {#if r.status === 'pending'}
                <button onclick={() => withdrawUrgentSub(r.requestId)} class="px-2.5 py-1 rounded bg-slate-800 text-rose-300 cursor-pointer">
                  Tarik
                </button>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/if}

<script lang="ts">
  import { apiFetch, app, navigate, type RouteKey } from '../lib/state.svelte';

  let query = $state('');
  let searchInput = $state<HTMLInputElement | null>(null);
  let results = $state<{ students: any[]; groups: any[]; tasks: any[] }>({
    students: [],
    groups: [],
    tasks: [],
  });

  $effect(() => {
    if (app.paletteOpen && searchInput) {
      searchInput.focus();
    }
  });

  const quickNav: { key: RouteKey; label: string; desc: string }[] = [
    { key: 'dashboard', label: 'Dasbor & KPI Saya', desc: '27 Grup aktif, kehadiran & Laporan ke TL' },
    { key: 'taskboard', label: 'Papan Tugas (Kanban)', desc: 'Tugas mingguan, SLA & Pengganti Mendesak' },
    { key: 'students', label: 'Daftar 55 Siswa', desc: 'Filter kuota kelas <= 3 & kontak WA Ortu' },
    { key: 'regular-feedback', label: 'Regular Feedback & PDF A4', desc: 'Generator 32 kursus Coding/Design & WA' },
    { key: 'schedule', label: 'Jadwal, Slot & Reschedule', desc: '217 slot ketersediaan & Reschedule Murid Private' },
    { key: 'assessment', label: 'Tutor Assessment', desc: 'Siklus aktif & grafik tren 37 siklus' },
    { key: 'surveys', label: 'Survei Siswa (Renewal)', desc: 'Evaluasi 9 langkah pra-perpanjangan 4 minggu' },
    { key: 'communication', label: 'Bantuan & Inbox Chat', desc: 'Basis Pengetahuan SOP & pesan ke Team Lead' },
    { key: 'tl-demo', label: 'Mode Demo Team Lead', desc: 'Simulasi persetujuan Reschedule & Laporan TL' },
    { key: 'benchmark', label: 'Benchmark Pitching Pimpinan', desc: 'Perbandingan kecepatan live vs aplikasi lama' },
  ];

  $effect(() => {
    const q = query.trim();
    if (q.length < 2) {
      results = { students: [], groups: [], tasks: [] };
      return;
    }
    const timer = setTimeout(async () => {
      try {
        results = await apiFetch(`/api/v1/search?q=${encodeURIComponent(q)}`);
      } catch {
        // ignore
      }
    }, 90);
    return () => clearTimeout(timer);
  });

  function jump(r: RouteKey, param = '') {
    app.paletteOpen = false;
    query = '';
    navigate(r, param);
  }
</script>

{#if app.paletteOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-start justify-center pt-20 p-4"
    onclick={(e) => {
      if (e.target === e.currentTarget) app.paletteOpen = false;
    }}
  >
    <div class="w-full max-w-xl bg-slate-900 border border-slate-700 rounded-2xl shadow-2xl overflow-hidden">
      <div class="p-3.5 border-b border-slate-800 flex items-center gap-3">
        <span class="text-slate-400 text-sm">🔎</span>
        <input
          bind:this={searchInput}
          bind:value={query}
          placeholder="Cari halaman, nama siswa, ID grup LMS (mis. 1710124), atau tugas..."
          class="flex-1 bg-transparent text-sm text-white focus:outline-none"
        />
        <button onclick={() => (app.paletteOpen = false)} class="text-xs px-2 py-1 rounded bg-slate-800 text-slate-400 cursor-pointer">
          ESC
        </button>
      </div>

      <div class="max-h-96 overflow-y-auto p-2 space-y-3">
        {#if query.trim().length < 2}
          <div class="px-2 py-1 text-[11px] font-semibold uppercase tracking-wider text-slate-500">Navigasi Cepat</div>
          {#each quickNav as item}
            <button
              onclick={() => jump(item.key)}
              class="w-full text-left px-3 py-2 rounded-xl hover:bg-slate-800 flex items-center justify-between transition cursor-pointer"
            >
              <div>
                <div class="text-sm font-medium text-slate-100">{item.label}</div>
                <div class="text-xs text-slate-400">{item.desc}</div>
              </div>
              <span class="text-xs text-indigo-400">&rarr;</span>
            </button>
          {/each}
        {:else}
          {#if results.students.length > 0}
            <div>
              <div class="px-2 py-1 text-[11px] font-semibold uppercase text-slate-500">Siswa ({results.students.length})</div>
              {#each results.students as st}
                <button
                  onclick={() => jump('students', st.studentId)}
                  class="w-full text-left px-3 py-2 rounded-xl hover:bg-slate-800 flex items-center justify-between cursor-pointer"
                >
                  <div>
                    <div class="text-sm font-medium text-white">{st.fullName}</div>
                    <div class="text-xs text-slate-400">{st.groupTitle} • Sisa kuota: {st.lessonsRemaining}</div>
                  </div>
                  <span class="text-xs text-emerald-400">{st.attendancePct}% hadir</span>
                </button>
              {/each}
            </div>
          {/if}

          {#if results.groups.length > 0}
            <div>
              <div class="px-2 py-1 text-[11px] font-semibold uppercase text-slate-500">Grup Kelas ({results.groups.length})</div>
              {#each results.groups as g}
                <button
                  onclick={() => jump('dashboard', g.groupId)}
                  class="w-full text-left px-3 py-2 rounded-xl hover:bg-slate-800 flex items-center justify-between cursor-pointer"
                >
                  <div>
                    <div class="text-sm font-medium text-white">{g.title}</div>
                    <div class="text-xs text-slate-400">LMS #{g.groupId} • {g.scheduleSummary}</div>
                  </div>
                  <span class="text-xs px-2 py-0.5 rounded bg-slate-800 text-indigo-300">{g.format}</span>
                </button>
              {/each}
            </div>
          {/if}

          {#if results.tasks.length > 0}
            <div>
              <div class="px-2 py-1 text-[11px] font-semibold uppercase text-slate-500">Tugas ({results.tasks.length})</div>
              {#each results.tasks as tk}
                <button
                  onclick={() => jump('taskboard', tk.taskId)}
                  class="w-full text-left px-3 py-2 rounded-xl hover:bg-slate-800 flex items-center justify-between cursor-pointer"
                >
                  <div class="text-sm text-white truncate pr-3">{tk.title}</div>
                  <span class="text-xs text-amber-400">{tk.status}</span>
                </button>
              {/each}
            </div>
          {/if}

          {#if results.students.length === 0 && results.groups.length === 0 && results.tasks.length === 0}
            <div class="p-6 text-center text-sm text-slate-500">Tidak ditemukan hasil untuk "{query}"</div>
          {/if}
        {/if}
      </div>
    </div>
  </div>
{/if}

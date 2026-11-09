<script lang="ts">
  import { apiFetch, app, buildWhatsAppURL, copyToClipboard, fmtDateTimeWIB, notify, refreshBootstrap } from '../lib/state.svelte';

  let students = $state<any[]>(app.cache.students || []);
  let search = $state('');
  let balanceFilter = $state<'all' | 'low' | 'normal' | 'high'>('all');
  let formatFilter = $state<'all' | 'private' | 'group'>('all');
  let courseFilter = $state('all');
  let selectedStudentDetail = $state<any>(null);
  let newViewName = $state('');

  async function loadStudents() {
    const res = await apiFetch('/api/v1/students');
    students = res.students || [];
    app.cache.students = students;
  }

  async function openStudent(id: string) {
    selectedStudentDetail = await apiFetch(`/api/v1/students/${id}`);
  }

  async function saveCurrentView() {
    if (!newViewName.trim()) return;
    const qs = `balance=${balanceFilter}&format=${formatFilter}&course=${courseFilter}`;
    await apiFetch('/api/v1/saved-views', {
      method: 'POST',
      body: JSON.stringify({ pageKey: 'students', name: newViewName, queryString: qs }),
    });
    newViewName = '';
    await refreshBootstrap();
    notify('Filter tersimpan ke Saved Views!', 'success');
  }

  function applySavedView(qs: string) {
    const p = new URLSearchParams(qs);
    balanceFilter = (p.get('balance') as any) || 'all';
    formatFilter = (p.get('format') as any) || 'all';
    courseFilter = p.get('course') || 'all';
  }

  $effect(() => {
    loadStudents();
    if (app.routeParam) {
      openStudent(app.routeParam);
      app.routeParam = '';
    }
  });

  $effect(() => {
    const handleSSE = (e: any) => {
      if (e.detail?.event === 'sync_completed') {
        loadStudents();
      }
    };
    window.addEventListener('ak-sse', handleSSE);
    return () => window.removeEventListener('ak-sse', handleSSE);
  });

  const coursesList = $derived.by(() => {
    const s = new Set<string>();
    students.forEach((st) => s.add(st.courseCode));
    return Array.from(s).sort();
  });

  const filtered = $derived.by(() => {
    const q = search.trim().toLowerCase();
    return students.filter((st) => {
      if (formatFilter !== 'all' && st.groupFormat !== formatFilter) return false;
      if (courseFilter !== 'all' && st.courseCode !== courseFilter) return false;
      if (balanceFilter === 'low' && st.lessonsRemaining > 3) return false;
      if (balanceFilter === 'normal' && (st.lessonsRemaining < 4 || st.lessonsRemaining > 9)) return false;
      if (balanceFilter === 'high' && st.lessonsRemaining < 10) return false;
      if (q && !st.fullName.toLowerCase().includes(q) && !st.groupTitle.toLowerCase().includes(q) && !st.studentId.includes(q)) {
        return false;
      }
      return true;
    });
  });

  const lowBalanceCount = $derived(students.filter((s) => s.lessonsRemaining <= 3).length);
</script>

<div class="space-y-4">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-bold text-white">Daftar Siswa & Monitoring Kuota ({students.length})</h2>
      <p class="text-xs text-slate-400">Pencarian cepat nama siswa, status kuota sesi pelajaran, kehadiran, dan kontak orang tua.</p>
    </div>

    <div class="flex items-center gap-2">
      {#each (app.savedViews || []).filter((v) => v.pageKey === 'students') as sv}
        <button
          onclick={() => applySavedView(sv.queryString)}
          class="px-3 py-1.5 rounded-xl bg-slate-900 border border-slate-800 hover:border-indigo-500 text-xs text-indigo-300 cursor-pointer"
        >
          ★ {sv.name}
        </button>
      {/each}
    </div>
  </div>

  <!-- KPI Summary Strip -->
  <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
    <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
      <div class="text-[11px] uppercase font-semibold text-slate-400">Siswa Ditampilkan</div>
      <div class="mt-1 text-2xl font-black text-white">{filtered.length} <span class="text-sm text-slate-500">/ {students.length}</span></div>
    </div>
    <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
      <div class="text-[11px] uppercase font-semibold text-slate-400">Kuota Kritis (&le; 3 Sesi)</div>
      <div class="mt-1 text-2xl font-black text-rose-400">{lowBalanceCount} siswa</div>
    </div>
    <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800 flex items-center gap-2">
      <input
        bind:value={newViewName}
        placeholder="Simpan kombinasi filter..."
        class="flex-1 px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"
      />
      <button onclick={saveCurrentView} class="px-3 py-2 rounded-xl bg-indigo-600 text-white text-xs font-semibold cursor-pointer">
        Simpan View
      </button>
    </div>
  </div>

  <!-- Filter Bar -->
  <div class="flex flex-wrap items-center gap-2.5 bg-slate-900 border border-slate-800 p-3 rounded-2xl">
    <input
      bind:value={search}
      placeholder="Cari nama siswa, grup, atau ID LMS..."
      class="flex-1 min-w-56 px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white focus:outline-none focus:border-indigo-500"
    />
    <select bind:value={balanceFilter} class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white">
      <option value="all">Semua Sisa Kuota</option>
      <option value="low">Kritis (&le; 3 Sesi)</option>
      <option value="normal">Normal (4–9 Sesi)</option>
      <option value="high">Aman (10+ Sesi)</option>
    </select>
    <select bind:value={formatFilter} class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white">
      <option value="all">Semua Format</option>
      <option value="private">Private 1-on-1</option>
      <option value="group">Grup Reguler</option>
    </select>
    <select bind:value={courseFilter} class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white">
      <option value="all">Semua Kursus</option>
      {#each coursesList as c}
        <option value={c}>{c}</option>
      {/each}
    </select>
  </div>

  <!-- Roster Table -->
  <div class="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead class="bg-slate-950 text-slate-400 uppercase text-[11px] border-b border-slate-800">
          <tr>
            <th class="py-3 px-4">Nama Siswa</th>
            <th class="py-3 px-3">Grup & Format</th>
            <th class="py-3 px-3">Kursus</th>
            <th class="py-3 px-3">Kehadiran</th>
            <th class="py-3 px-3">Sisa Kuota Kelas</th>
            <th class="py-3 px-4 text-right">Kontak Orang Tua</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/70">
          {#each filtered as st (st.studentId)}
            <tr onclick={() => openStudent(st.studentId)} class="hover:bg-slate-800/60 transition cursor-pointer">
              <td class="py-3 px-4">
                <div class="font-semibold text-white text-sm">{st.fullName}</div>
                <div class="text-[11px] text-slate-500">LMS #{st.studentId}</div>
              </td>
              <td class="py-3 px-3">
                <div class="text-slate-200 font-medium">{st.groupTitle}</div>
                <span class="text-[10px] px-1.5 py-0.5 rounded {st.groupFormat === 'private' ? 'bg-amber-500/15 text-amber-300' : 'bg-slate-800 text-slate-400'}">
                  {st.groupFormat === 'private' ? 'Private 1-1' : 'Grup'}
                </span>
              </td>
              <td class="py-3 px-3 text-indigo-300 font-semibold">{st.courseCode}</td>
              <td class="py-3 px-3">
                <span class="font-bold {st.attendancePct >= 85 ? 'text-emerald-400' : 'text-amber-400'}">{st.attendancePct}%</span>
                <span class="text-slate-500"> ({st.lessonsAttended}/{st.lessonsTotal})</span>
              </td>
              <td class="py-3 px-3">
                <span class="px-2.5 py-0.5 rounded-full font-bold {st.lessonsRemaining <= 3 ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30' : st.lessonsRemaining <= 9 ? 'bg-amber-500/15 text-amber-300' : 'bg-emerald-500/15 text-emerald-300'}">
                  {st.lessonsRemaining} / {st.totalLessonsPurchased}
                </span>
              </td>
              <td class="py-3 px-4 text-right" onclick={(e) => e.stopPropagation()}>
                <button
                  onclick={() => copyToClipboard(st.parentPhone, 'Nomor WA Orang Tua')}
                  class="px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 mr-1 cursor-pointer"
                >
                  Salin
                </button>
                <a
                  href={buildWhatsAppURL(st.parentPhone, `Halo Ayah/Bunda dari ${st.fullName}, salam kenal dari ${app.tutor?.firstName ? 'Kak ' + app.tutor.firstName : 'Tutor'} (Pengajar Algonova Indonesia).`)}
                  target="_blank"
                  rel="noreferrer"
                  class="px-2.5 py-1 rounded-lg bg-emerald-600/20 text-emerald-300 hover:bg-emerald-600/35 font-semibold"
                >
                  WA Ortu
                </a>
              </td>
            </tr>
          {/each}

          {#if filtered.length === 0}
            <tr>
              <td colspan="6" class="py-10 text-center text-xs text-slate-500">
                Belum ada data siswa terdaftar.
              </td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  </div>
</div>

{#if selectedStudentDetail}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-40 bg-black/65 flex justify-end" onclick={(e) => e.target === e.currentTarget && (selectedStudentDetail = null)}>
    <div class="w-full max-w-lg bg-slate-900 border-l border-slate-800 h-full overflow-y-auto p-6 space-y-4">
      <div class="flex items-start justify-between">
        <div>
          <h3 class="text-lg font-bold text-white">{selectedStudentDetail.student.fullName}</h3>
          <p class="text-xs text-slate-400">{selectedStudentDetail.student.groupTitle} • LMS #{selectedStudentDetail.student.studentId}</p>
        </div>
        <button onclick={() => (selectedStudentDetail = null)} class="px-2.5 py-1 rounded bg-slate-800 text-xs text-slate-300 cursor-pointer">✕</button>
      </div>

      <div class="grid grid-cols-2 gap-3 text-xs">
        <div class="p-3 rounded-xl bg-slate-950 border border-slate-800">
          <div class="text-slate-500">Kehadiran</div>
          <div class="text-lg font-bold text-emerald-400">{selectedStudentDetail.student.attendancePct}%</div>
        </div>
        <div class="p-3 rounded-xl bg-slate-950 border border-slate-800">
          <div class="text-slate-500">Sisa Saldo Kelas</div>
          <div class="text-lg font-bold text-white">{selectedStudentDetail.student.lessonsRemaining} sesi</div>
        </div>
      </div>

      <div>
        <h4 class="text-xs font-bold uppercase text-slate-400 mb-2">Riwayat Regular Feedback Tersimpan</h4>
        {#if (selectedStudentDetail.feedbackReports || []).length === 0}
          <div class="p-4 rounded-xl bg-slate-950 border border-slate-800 text-xs text-slate-500">Belum ada Regular Feedback tersimpan untuk siswa ini.</div>
        {:else}
          <div class="space-y-2.5">
            {#each selectedStudentDetail.feedbackReports as rp}
              <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800 text-xs space-y-2">
                <div class="flex items-center justify-between">
                  <span class="font-bold text-indigo-300">Pertemuan #{rp.lessonNumber} • Bulan {rp.monthNo}</span>
                  <a href="/api/v1/feedback/regular/{rp.reportId}/print" target="_blank" class="text-emerald-400 font-semibold">Cetak PDF A4 &nearr;</a>
                </div>
                <p class="text-slate-300 whitespace-pre-line line-clamp-4">{rp.assembledText}</p>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

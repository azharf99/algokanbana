<script lang="ts">
  import { apiFetch, app, buildWhatsAppURL, copyToClipboard, fmtDateTimeWIB, notify } from '../lib/state.svelte';

  let modeTab = $state<'coding' | 'mt'>('coding');
  let weekParam = $state('');
  let fbData = $state<any>(app.cache.regularFeedback || null);
  let mtEntries = $state<any[]>([]);

  // Composer Drawer State
  let drawerOpen = $state(false);
  let activeLesson = $state<any>(null);
  let activeStudent = $state<any>(null);
  let courseCode = $state('PS');
  let monthNo = $state(1);
  let attendance = $state('full');
  let engagement = $state('very_active');
  let tasks = $state('all_done');
  let custom1 = $state('');
  let custom2 = $state('');
  let projectUrl = $state('');
  let stars = $state(5);
  let assembledText = $state('');
  let savedReportId = $state('');
  let saving = $state(false);

  // MT Form State
  let mtStudentName = $state('Kenzie Pratama');
  let mtTopic = $state('');
  let mtUnderstanding = $state(5);
  let mtIndependence = $state(5);
  let mtAccuracy = $state(5);
  let mtEngagement = $state('active');
  let mtComment = $state('');

  async function loadWeek(week = weekParam) {
    const q = week ? `?week=${encodeURIComponent(week)}` : '';
    const d = await apiFetch(`/api/v1/feedback/regular${q}`);
    fbData = d;
    weekParam = d.weekStart;
    app.cache.regularFeedback = d;
  }

  async function loadMT() {
    const res = await apiFetch('/api/v1/feedback/mt');
    mtEntries = res.entries || [];
  }

  async function refreshPreview() {
    if (!activeStudent) return;
    try {
      const res = await apiFetch('/api/v1/feedback/regular/preview', {
        method: 'POST',
        body: JSON.stringify({
          locale: 'id',
          studentName: activeStudent.studentName,
          courseCode,
          monthNo,
          attendance,
          engagement,
          tasks,
          custom1,
          custom2,
          projectUrl,
          stars,
        }),
      });
      assembledText = res.assembledText;
    } catch {
      // ignore
    }
  }

  function openComposer(lesson: any, student: any) {
    activeLesson = lesson;
    activeStudent = student;

    const existing = (fbData?.reports || []).find(
      (r: any) => r.lmsLessonId === lesson.lessonId && r.lmsStudentId === student.studentId
    );
    const draftKey = `nk.feedback.draft.${lesson.lessonId}:${student.studentId}`;
    const rawDraft = localStorage.getItem(draftKey);
    const draft = rawDraft ? JSON.parse(rawDraft) : null;

    if (existing) {
      courseCode = existing.courseCode;
      monthNo = existing.monthNo;
      attendance = existing.attendance;
      engagement = existing.engagement;
      tasks = existing.tasks;
      custom1 = existing.custom1;
      custom2 = existing.custom2;
      projectUrl = existing.projectUrl;
      stars = existing.stars;
      assembledText = existing.assembledText;
      savedReportId = existing.reportId;
    } else if (draft?.values) {
      courseCode = draft.values.courseCode || lesson.courseCode || 'PS';
      monthNo = draft.values.monthNo || Math.max(1, Math.ceil((lesson.courseLessonNumber || 4) / 4));
      attendance = draft.values.attendance || 'full';
      engagement = draft.values.engagement || 'very_active';
      tasks = draft.values.tasks || 'all_done';
      custom1 = draft.values.custom1 || '';
      custom2 = draft.values.custom2 || '';
      projectUrl = draft.values.projectUrl || '';
      stars = draft.values.stars || 5;
      savedReportId = '';
      refreshPreview();
    } else {
      courseCode = lesson.courseCode || 'PS';
      monthNo = Math.max(1, Math.ceil((lesson.courseLessonNumber || 4) / 4));
      attendance = 'full';
      engagement = 'very_active';
      tasks = 'all_done';
      custom1 = '';
      custom2 = '';
      projectUrl = '';
      stars = 5;
      savedReportId = '';
      refreshPreview();
    }
    drawerOpen = true;
  }

  function persistDraft() {
    if (!activeLesson || !activeStudent) return;
    const draftKey = `nk.feedback.draft.${activeLesson.lessonId}:${activeStudent.studentId}`;
    localStorage.setItem(
      draftKey,
      JSON.stringify({
        savedAt: Date.now(),
        values: { courseCode, monthNo, attendance, engagement, tasks, custom1, custom2, projectUrl, stars },
      })
    );
    refreshPreview();
  }

  async function saveFeedback(e: Event) {
    e.preventDefault();
    if (!activeLesson || !activeStudent) return;
    saving = true;
    try {
      const res = await apiFetch('/api/v1/feedback/regular', {
        method: 'PUT',
        body: JSON.stringify({
          weekStart: fbData.weekStart,
          lmsLessonId: activeLesson.lessonId,
          lmsStudentId: activeStudent.studentId,
          studentName: activeStudent.studentName,
          parentPhone: activeStudent.parentPhone,
          groupTitle: activeLesson.groupTitle,
          lessonNumber: activeLesson.courseLessonNumber,
          courseCode,
          monthNo,
          attendance,
          engagement,
          tasks,
          custom1,
          custom2,
          stars,
          projectUrl,
          assembledText,
        }),
      });
      savedReportId = res.report.reportId;
      assembledText = res.report.assembledText;
      localStorage.removeItem(`nk.feedback.draft.${activeLesson.lessonId}:${activeStudent.studentId}`);
      await loadWeek(fbData.weekStart);
      notify('Regular Feedback tersimpan! Siap cetak PDF A4 atau kirim WA Ortu.', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    } finally {
      saving = false;
    }
  }

  async function submitMTEntry(e: Event) {
    e.preventDefault();
    try {
      await apiFetch('/api/v1/feedback/mt', {
        method: 'POST',
        body: JSON.stringify({
          weekStart: fbData?.weekStart || '2026-10-19',
          lmsLessonId: '9500999',
          lmsStudentId: '880101',
          studentName: mtStudentName,
          lessonNumber: 4,
          lessonTopic: mtTopic,
          understanding: mtUnderstanding,
          independence: mtIndependence,
          accuracy: mtAccuracy,
          engagement: mtEngagement,
          tutorComment: mtComment,
        }),
      });
      mtTopic = '';
      mtComment = '';
      await loadMT();
      notify('Feedback MT berhasil disimpan!', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  $effect(() => {
    loadWeek('');
  });

  const progressPct = $derived(
    fbData?.totalTargets > 0 ? Math.round((fbData.filledTargets * 100) / fbData.totalTargets) : 100
  );
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-bold text-white">Regular Feedback & Laporan Orang Tua</h2>
      <p class="text-xs text-slate-400">Milestone otomatis pertemuan #4, #8, #12, #16 — lengkap dengan template 32 kursus, ekspor PDF A4, & WA 1-klik.</p>
    </div>

    <div class="inline-flex rounded-xl bg-slate-900 border border-slate-800 p-1">
      <button
        onclick={() => (modeTab = 'coding')}
        class="px-3.5 py-1.5 rounded-lg text-xs font-semibold cursor-pointer {modeTab === 'coding' ? 'bg-indigo-600 text-white' : 'text-slate-400'}"
      >
        Regular Feedback (Coding/Design)
      </button>
      <button
        onclick={() => {
          modeTab = 'mt';
          loadMT();
        }}
        class="px-3.5 py-1.5 rounded-lg text-xs font-semibold cursor-pointer {modeTab === 'mt' ? 'bg-indigo-600 text-white' : 'text-slate-400'}"
      >
        Feedback MT (Math)
      </button>
    </div>
  </div>

  {#if modeTab === 'coding'}
    <!-- Week Navigator & Progress Card -->
    <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800 flex flex-wrap items-center justify-between gap-4">
      <div class="flex items-center gap-2">
        <button onclick={() => loadWeek(fbData?.prevWeek)} class="px-3 py-1.5 rounded-xl bg-slate-950 border border-slate-800 text-xs text-slate-300 cursor-pointer">
          &larr; Pekan Lalu
        </button>
        <div class="px-3.5 py-1.5 rounded-xl bg-indigo-600/15 border border-indigo-500/30 text-xs font-bold text-indigo-300">
          Pekan Mulai: {fbData?.weekStart || '...'}
        </div>
        <button onclick={() => loadWeek(fbData?.nextWeek)} class="px-3 py-1.5 rounded-xl bg-slate-950 border border-slate-800 text-xs text-slate-300 cursor-pointer">
          Pekan Depan &rarr;
        </button>
      </div>

      <div class="flex-1 min-w-64 max-w-md">
        <div class="flex justify-between text-xs mb-1">
          <span class="text-slate-400">Progres Pekan Ini (Tenggat Minggu 23:59 WIB)</span>
          <span class="font-bold text-emerald-400">{fbData?.filledTargets ?? 0} / {fbData?.totalTargets ?? 0} ({progressPct}%)</span>
        </div>
        <div class="w-full h-2 rounded-full bg-slate-950 overflow-hidden">
          <div class="h-full bg-emerald-500 transition-all" style="width: {progressPct}%"></div>
        </div>
      </div>
    </div>

    <!-- Milestone Cards -->
    {#if (fbData?.milestones || []).length === 0}
      <div class="p-8 rounded-2xl bg-slate-900 border border-slate-800 text-center text-sm text-slate-400">
        Tidak ada pertemuan kelipatan ke-4 (#4, #8, #12, #16) pada pekan ini. Gunakan tombol navigasi pekan di atas.
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        {#each fbData.milestones as ls (ls.lessonId)}
          <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800 space-y-3">
            <div class="flex items-start justify-between gap-2">
              <div>
                <span class="text-[11px] font-bold px-2 py-0.5 rounded bg-indigo-600/20 text-indigo-300">
                  Pertemuan #{ls.courseLessonNumber} • {ls.courseCode}
                </span>
                <h3 class="mt-1.5 text-sm font-bold text-white">{ls.groupTitle}</h3>
                <div class="text-xs text-slate-400">{fmtDateTimeWIB(ls.startsAt)}</div>
              </div>
              <span class="text-[11px] px-2 py-0.5 rounded bg-slate-800 text-slate-300">{ls.format}</span>
            </div>

            <div class="space-y-2 pt-2 border-t border-slate-800">
              {#each ls.students || [] as st}
                <div class="flex items-center justify-between p-2.5 rounded-xl bg-slate-950 border border-slate-800 text-xs">
                  <div>
                    <div class="font-semibold text-white">{st.studentName}</div>
                    <div class="text-[11px] {st.reportFilled ? 'text-emerald-400' : 'text-amber-400'}">
                      {st.reportFilled ? '✓ Umpan balik terisi' : '• Perlu diisi'}
                    </div>
                  </div>
                  <div class="flex items-center gap-1.5">
                    {#if st.reportFilled && st.reportId}
                      <a
                        href="/api/v1/feedback/regular/{st.reportId}/print"
                        target="_blank"
                        class="px-2.5 py-1 rounded-lg bg-emerald-600/20 text-emerald-300 font-semibold"
                      >
                        PDF A4
                      </a>
                    {/if}
                    <button
                      onclick={() => openComposer(ls, st)}
                      class="px-3 py-1 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold cursor-pointer"
                    >
                      {st.reportFilled ? 'Edit / WA' : 'Susun Feedback'}
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {:else}
    <!-- Feedback MT Tab -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <form onsubmit={submitMTEntry} class="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-3.5">
        <h3 class="text-sm font-bold text-white">Input Evaluasi Pertemuan Math (Feedback MT)</h3>
        <input bind:value={mtStudentName} required placeholder="Nama Siswa" class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white" />
        <input bind:value={mtTopic} required placeholder="Topik Pelajaran (wajib)" class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white" />
        <div class="grid grid-cols-3 gap-2 text-xs">
          <label class="p-2.5 rounded-xl bg-slate-950 border border-slate-800">
            <span class="block text-slate-400 mb-1">Pemahaman (1-5)</span>
            <input type="number" min="1" max="5" bind:value={mtUnderstanding} class="w-full bg-transparent text-white font-bold" />
          </label>
          <label class="p-2.5 rounded-xl bg-slate-950 border border-slate-800">
            <span class="block text-slate-400 mb-1">Kemandirian (1-5)</span>
            <input type="number" min="1" max="5" bind:value={mtIndependence} class="w-full bg-transparent text-white font-bold" />
          </label>
          <label class="p-2.5 rounded-xl bg-slate-950 border border-slate-800">
            <span class="block text-slate-400 mb-1">Ketelitian (1-5)</span>
            <input type="number" min="1" max="5" bind:value={mtAccuracy} class="w-full bg-transparent text-white font-bold" />
          </label>
        </div>
        <textarea bind:value={mtComment} rows="3" placeholder="Catatan perkembangan & langkah selanjutnya..." class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"></textarea>
        <button type="submit" class="w-full py-2.5 rounded-xl bg-indigo-600 text-white text-xs font-semibold cursor-pointer">Simpan Evaluasi MT</button>
      </form>

      <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-3">
        <h3 class="text-sm font-bold text-white">Riwayat Entri Feedback MT ({mtEntries.length})</h3>
        {#if mtEntries.length === 0}
          <p class="text-xs text-slate-500">Belum ada entri Feedback MT pada sesi lokal ini.</p>
        {:else}
          {#each mtEntries as e}
            <div class="p-3 rounded-xl bg-slate-950 border border-slate-800 text-xs">
              <div class="font-bold text-white">{e.studentName} — {e.lessonTopic}</div>
              <div class="text-emerald-400 mt-0.5">Skor: Pemahaman {e.understanding}/5 • Mandiri {e.independence}/5 • Teliti {e.accuracy}/5</div>
              {#if e.tutorComment}<p class="text-slate-400 mt-1">{e.tutorComment}</p>{/if}
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/if}
</div>

<!-- Regular Feedback Composer Drawer -->
{#if drawerOpen && activeStudent}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 bg-black/75 flex justify-end" onclick={(e) => e.target === e.currentTarget && (drawerOpen = false)}>
    <form onsubmit={saveFeedback} class="w-full max-w-2xl bg-slate-900 border-l border-slate-800 h-full overflow-y-auto p-6 space-y-4">
      <div class="flex items-start justify-between">
        <div>
          <span class="text-[11px] font-bold px-2 py-0.5 rounded bg-indigo-600/20 text-indigo-300">
            Pertemuan #{activeLesson?.courseLessonNumber} • Auto-Draft Aktif
          </span>
          <h3 class="mt-1 text-lg font-bold text-white">Regular Feedback — {activeStudent.studentName}</h3>
          <p class="text-xs text-slate-400">{activeLesson?.groupTitle}</p>
        </div>
        <button type="button" onclick={() => (drawerOpen = false)} class="px-2.5 py-1 rounded bg-slate-800 text-xs text-slate-300 cursor-pointer">✕</button>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
        <div class="sm:col-span-2">
          <label for="fb-course" class="block font-semibold text-slate-300 mb-1">Kursus Kurikulum (32 Kursus)</label>
          <select id="fb-course" bind:value={courseCode} onchange={persistDraft} class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white">
            {#each fbData?.courses || [] as c}
              <option value={c.code}>[{c.code}] {c.name} ({c.months} bln)</option>
            {/each}
          </select>
        </div>
        <div>
          <label for="fb-month" class="block font-semibold text-slate-300 mb-1">Bulan Belajar</label>
          <input id="fb-month" type="number" min="1" max="12" bind:value={monthNo} oninput={persistDraft} class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
        <div>
          <label for="fb-att" class="block font-semibold text-slate-300 mb-1">Kehadiran</label>
          <select id="fb-att" bind:value={attendance} onchange={persistDraft} class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white">
            <option value="full">Hadir Penuh</option>
            <option value="most">Sebagian Besar Hadir</option>
            <option value="half">Sekitar Separuh</option>
            <option value="low">Kehadiran Rendah</option>
          </select>
        </div>
        <div>
          <label for="fb-eng" class="block font-semibold text-slate-300 mb-1">Keaktifan Kelas</label>
          <select id="fb-eng" bind:value={engagement} onchange={persistDraft} class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white">
            <option value="very_active">Sangat Aktif & Antusias</option>
            <option value="focused_quiet">Fokus & Tenang</option>
            <option value="passive">Perlu Lebih Didorong</option>
            <option value="struggling">Butuh Bimbingan Bertahap</option>
          </select>
        </div>
        <div>
          <label for="fb-tsk" class="block font-semibold text-slate-300 mb-1">Penyelesaian Tugas</label>
          <select id="fb-tsk" bind:value={tasks} onchange={persistDraft} class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white">
            <option value="all_done">Semua Tugas Tuntas</option>
            <option value="mostly_done">Sebagian Besar Selesai</option>
            <option value="struggling">Perlu Latihan Tambahan</option>
          </select>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
        <div>
          <label for="fb-c1" class="block font-semibold text-slate-300 mb-1">Pencapaian Menonjol Siswa</label>
          <textarea id="fb-c1" bind:value={custom1} oninput={persistDraft} rows="2" placeholder="Contoh: Berhasil membuat mekanik leaderboard di Lua..." class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white"></textarea>
        </div>
        <div>
          <label for="fb-c2" class="block font-semibold text-slate-300 mb-1">Rekomendasi Bulan Berikutnya</label>
          <textarea id="fb-c2" bind:value={custom2} oninput={persistDraft} rows="2" placeholder="Contoh: Latihan merapikan fungsi & struktur kondisi..." class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white"></textarea>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
        <div class="sm:col-span-2">
          <label for="fb-url" class="block font-semibold text-slate-300 mb-1">Link Karya / Proyek Siswa (Opsional)</label>
          <input id="fb-url" bind:value={projectUrl} oninput={persistDraft} placeholder="https://scratch.mit.edu/... atau GitHub / Roblox" class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
        </div>
        <div>
          <label for="fb-stars" class="block font-semibold text-slate-300 mb-1">Bintang Apresiasi</label>
          <select id="fb-stars" bind:value={stars} onchange={persistDraft} class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-amber-400 font-bold">
            <option value={5}>★★★★★ (5/5)</option>
            <option value={4}>★★★★☆ (4/5)</option>
            <option value={3}>★★★☆☆ (3/5)</option>
          </select>
        </div>
      </div>

      <div>
        <label for="fb-prev" class="block text-xs font-semibold text-indigo-300 mb-1">Pratinjau Pesan Orang Tua (Dapat Diedit Langsung)</label>
        <textarea id="fb-prev" bind:value={assembledText} rows="9" class="w-full p-3.5 rounded-xl bg-slate-950 border border-indigo-500/30 text-xs text-slate-200 leading-relaxed"></textarea>
      </div>

      <div class="flex flex-wrap items-center gap-2 pt-2">
        <button type="submit" disabled={saving} class="px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer">
          {saving ? 'Menyimpan...' : '💾 Simpan Laporan'}
        </button>
        <button type="button" onclick={() => copyToClipboard(assembledText, 'Teks Feedback')} class="px-3.5 py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold cursor-pointer">
          📋 Salin Teks
        </button>
        <a
          href={buildWhatsAppURL(activeStudent.parentPhone || '', assembledText)}
          target="_blank"
          rel="noreferrer"
          class="px-3.5 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold"
        >
          📲 Kirim ke WhatsApp Ortu
        </a>
        {#if savedReportId}
          <a
            href="/api/v1/feedback/regular/{savedReportId}/print"
            target="_blank"
            rel="noreferrer"
            class="px-3.5 py-2.5 rounded-xl bg-amber-500/20 border border-amber-500/40 text-amber-300 text-xs font-semibold"
          >
            🖨️ Cetak / PDF A4
          </a>
        {/if}
      </div>
    </form>
  </div>
{/if}

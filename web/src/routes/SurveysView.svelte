<script lang="ts">
  import { apiFetch, fmtDateTimeWIB, notify } from '../lib/state.svelte';

  interface SurveySubmission {
    submissionId: string;
    surveyId: string;
    tutorId: string;
    studentId: string;
    nominationBadge: string;
    observationNotes: string;
    comprehensionScore: number;
    independenceScore: number;
    focusScore: number;
    stabilityScore: number;
    motivationScore: number;
    primaryMotivator: string;
    parentFeedbackNotes: string;
    submittedAt: string;
  }

  interface SurveyTarget {
    surveyId: string;
    tutorId: string;
    studentId: string;
    studentName: string;
    groupTitle: string;
    courseName: string;
    lessonsRemaining: number;
    deadlineAt: string;
    status: 'pending' | 'completed';
    createdAt: string;
    submission?: SurveySubmission;
  }

  let targets = $state<SurveyTarget[]>([]);
  let loading = $state(true);
  let activeTab = $state<'pending' | 'completed'>('pending');

  // Wizard state
  let wizardOpen = $state(false);
  let activeTarget = $state<SurveyTarget | null>(null);
  let currentStep = $state(1); // 1 to 9
  let viewOnlySubmission = $state<SurveySubmission | null>(null);

  // Form fields
  let nominationBadge = $state('Pemecah Masalah Terbaik');
  let observationNotes = $state('');
  let comprehensionScore = $state(4);
  let independenceScore = $state(4);
  let focusScore = $state(4);
  let stabilityScore = $state(4);
  let motivationScore = $state(5);
  let primaryMotivator = $state('');
  let parentFeedbackNotes = $state('');
  let submitting = $state(false);

  const badgesList = [
    '🏆 Pemecah Masalah Terbaik (Best Problem Solver)',
    '💡 Kreator Algoritma Terinovatif (Creative Thinker)',
    '🤝 Kolaborator Teraktif (Active Collaborator)',
    '🔥 Siswa Paling Gigih & Pantang Menyerah (Persistent)',
    '🎮 Arsitek Game Terbaik (Best Game Architect)',
    '🐞 Penemu Bug Tercepat (Master Debugger)',
  ];

  const motivatorSuggestions = [
    'Sangat antusias ingin menerbitkan game ciptaannya ke Roblox / Web',
    'Tertarik mempelajari Python dan Data Science tingkat lanjut',
    'Ingin persiapan kompetisi koding / olimpiade informatika',
    'Didukung penuh orang tua untuk portofolio beasiswa masa depan',
    'Senang berkarya bersama teman sekelas dan ingin lanjut level berikutnya',
  ];

  async function loadSurveys() {
    loading = true;
    try {
      const res = await apiFetch<{ targets?: SurveyTarget[]; surveys?: SurveyTarget[] }>('/api/v1/surveys');
      targets = res.targets || res.surveys || [];
    } catch (err: any) {
      notify(err.message, 'error');
    } finally {
      loading = false;
    }
  }

  function openWizard(t: SurveyTarget) {
    activeTarget = t;
    viewOnlySubmission = null;
    currentStep = 1;

    // Try load draft from localStorage
    const saved = localStorage.getItem(`ak_survey_draft_${t.surveyId}`);
    if (saved) {
      try {
        const d = JSON.parse(saved);
        nominationBadge = d.nominationBadge || badgesList[0];
        observationNotes = d.observationNotes || '';
        comprehensionScore = d.comprehensionScore || 4;
        independenceScore = d.independenceScore || 4;
        focusScore = d.focusScore || 4;
        stabilityScore = d.stabilityScore || 4;
        motivationScore = d.motivationScore || 5;
        primaryMotivator = d.primaryMotivator || '';
        parentFeedbackNotes = d.parentFeedbackNotes || '';
        notify('Draf evaluasi sebelumnya dipulihkan otomatis', 'info');
      } catch {
        // ignore parse error
      }
    } else {
      nominationBadge = badgesList[0];
      observationNotes = '';
      comprehensionScore = 4;
      independenceScore = 4;
      focusScore = 4;
      stabilityScore = 4;
      motivationScore = 5;
      primaryMotivator = '';
      parentFeedbackNotes = '';
    }
    wizardOpen = true;
  }

  function openSummary(t: SurveyTarget) {
    activeTarget = t;
    viewOnlySubmission = t.submission || null;
    wizardOpen = true;
  }

  function saveDraft() {
    if (!activeTarget) return;
    const payload = {
      nominationBadge,
      observationNotes,
      comprehensionScore,
      independenceScore,
      focusScore,
      stabilityScore,
      motivationScore,
      primaryMotivator,
      parentFeedbackNotes,
    };
    localStorage.setItem(`ak_survey_draft_${activeTarget.surveyId}`, JSON.stringify(payload));
  }

  function nextStep() {
    saveDraft();
    if (currentStep < 9) {
      currentStep++;
    }
  }

  function prevStep() {
    saveDraft();
    if (currentStep > 1) {
      currentStep--;
    }
  }

  async function submitSurvey() {
    if (!activeTarget) return;
    if (!observationNotes.trim()) {
      notify('Catatan observasi siswa wajib diisi!', 'error');
      currentStep = 2;
      return;
    }
    if (!primaryMotivator.trim()) {
      notify('Motivator utama siswa wajib diisi!', 'error');
      currentStep = 8;
      return;
    }

    submitting = true;
    try {
      await apiFetch(`/api/v1/surveys/${activeTarget.surveyId}/submit`, {
        method: 'POST',
        body: JSON.stringify({
          nominationBadge,
          observationNotes,
          comprehensionScore,
          independenceScore,
          focusScore,
          stabilityScore,
          motivationScore,
          primaryMotivator,
          parentFeedbackNotes,
        }),
      });

      localStorage.removeItem(`ak_survey_draft_${activeTarget.surveyId}`);
      notify(`Survei evaluasi ${activeTarget.studentName} berhasil dikirim!`, 'success');
      wizardOpen = false;
      activeTarget = null;
      await loadSurveys();
    } catch (err: any) {
      notify(err.message, 'error');
    } finally {
      submitting = false;
    }
  }

  const pendingTargets = $derived(targets.filter((t) => t.status === 'pending'));
  const completedTargets = $derived(targets.filter((t) => t.status === 'completed'));

  $effect(() => {
    loadSurveys();
  });
</script>

<div class="space-y-5">
  <!-- Header -->
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-bold text-white flex items-center gap-2">
        <span>📋 Survei Siswa & Evaluasi Perpanjangan (Renewal)</span>
      </h2>
      <p class="text-xs text-slate-400 mt-0.5">
        Evaluasi pra-renewal 4 minggu sebelum paket belajar siswa berakhir untuk rekomendasi kurikulum lanjutan Algonova.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={loadSurveys}
        class="px-3.5 py-2 rounded-xl bg-slate-900 border border-slate-800 hover:border-slate-700 text-xs text-slate-300 font-semibold cursor-pointer"
      >
        ↻ Segarkan
      </button>
    </div>
  </div>

  <!-- Tabs Navigation -->
  <div class="flex items-center gap-2 border-b border-slate-800 pb-2 text-xs font-semibold">
    <button
      onclick={() => (activeTab = 'pending')}
      class="px-3.5 py-1.5 rounded-xl cursor-pointer transition {activeTab === 'pending'
        ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
        : 'text-slate-400 hover:text-white'}"
    >
      Perlu Diisi ({pendingTargets.length})
    </button>
    <button
      onclick={() => (activeTab = 'completed')}
      class="px-3.5 py-1.5 rounded-xl cursor-pointer transition {activeTab === 'completed'
        ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40'
        : 'text-slate-400 hover:text-white'}"
    >
      Selesai ({completedTargets.length})
    </button>
  </div>

  <!-- Content List -->
  {#if loading}
    <div class="p-8 text-center text-xs text-slate-500">Memuat target survei siswa...</div>
  {:else}
    {@const currentList = activeTab === 'pending' ? pendingTargets : completedTargets}
    {#if currentList.length === 0}
      <div class="p-8 text-center bg-slate-900/50 border border-slate-800/80 rounded-2xl text-xs text-slate-400">
        {#if activeTab === 'pending'}
          🎉 Hebat! Tidak ada survei siswa pra-renewal yang tertunda saat ini.
        {:else}
          Belum ada riwayat survei yang diselesaikan.
        {/if}
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        {#each currentList as t (t.surveyId)}
          <div class="p-4 rounded-2xl bg-slate-900/90 border border-slate-800 flex flex-col justify-between space-y-3">
            <div>
              <div class="flex items-start justify-between">
                <div>
                  <div class="text-sm font-bold text-white">{t.studentName}</div>
                  <div class="text-[11px] text-slate-400 mt-0.5">{t.groupTitle} • {t.courseName}</div>
                </div>
                {#if t.status === 'pending'}
                  <span class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30">
                    {t.lessonsRemaining} Sesi Tersisa
                  </span>
                {:else}
                  <span class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                    ✓ Selesai
                  </span>
                {/if}
              </div>

              <div class="mt-3 p-2.5 rounded-xl bg-slate-950 border border-slate-800/80 text-[11px] space-y-1">
                <div class="flex items-center justify-between text-slate-400">
                  <span>Tenggat Pengisian:</span>
                  <span class="font-semibold text-slate-200">{fmtDateTimeWIB(t.deadlineAt)}</span>
                </div>
                {#if t.submission}
                  <div class="flex items-center justify-between text-slate-400">
                    <span>Nominasi:</span>
                    <span class="font-semibold text-indigo-300 truncate max-w-xs">{t.submission.nominationBadge}</span>
                  </div>
                {/if}
              </div>
            </div>

            <div class="pt-2 border-t border-slate-800 flex items-center justify-end">
              {#if t.status === 'pending'}
                <button
                  onclick={() => openWizard(t)}
                  class="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer transition shadow-sm shadow-indigo-600/30"
                >
                  Mulai Evaluasi (9 Langkah) →
                </button>
              {:else}
                <button
                  onclick={() => openSummary(t)}
                  class="px-3.5 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold cursor-pointer transition"
                >
                  Lihat Jawaban Evaluasi
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<!-- 9-Step Wizard & Summary Modal -->
{#if wizardOpen && activeTarget}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4" onclick={(e) => e.target === e.currentTarget && (wizardOpen = false)}>
    <div class="w-full max-w-2xl bg-slate-900 border border-slate-800 rounded-3xl p-6 space-y-5 max-h-[90vh] overflow-y-auto">
      {#if viewOnlySubmission}
        <!-- Read-Only Summary View -->
        <div class="flex items-start justify-between border-b border-slate-800 pb-3">
          <div>
            <span class="text-[10px] font-bold uppercase px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300">
              Evaluasi Telah Disubmit
            </span>
            <h3 class="text-lg font-bold text-white mt-1">Rangkuman Evaluasi: {activeTarget.studentName}</h3>
            <div class="text-xs text-slate-400">{activeTarget.groupTitle} • Disubmit pada {fmtDateTimeWIB(viewOnlySubmission.submittedAt)}</div>
          </div>
          <button onclick={() => (wizardOpen = false)} class="px-2.5 py-1 rounded-lg bg-slate-800 text-xs text-slate-300 cursor-pointer">✕</button>
        </div>

        <div class="space-y-3.5 text-xs">
          <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
            <div class="text-slate-400">Penghargaan / Nominasi Siswa:</div>
            <div class="text-sm font-bold text-indigo-300 mt-1">{viewOnlySubmission.nominationBadge}</div>
          </div>

          <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
            <div class="text-slate-400">Observasi & Catatan Perkembangan:</div>
            <div class="text-slate-200 mt-1 whitespace-pre-wrap">{viewOnlySubmission.observationNotes}</div>
          </div>

          <div class="grid grid-cols-2 sm:grid-cols-5 gap-2">
            {#each [
              ['Pemahaman', viewOnlySubmission.comprehensionScore],
              ['Kemandirian', viewOnlySubmission.independenceScore],
              ['Fokus', viewOnlySubmission.focusScore],
              ['Stabilitas', viewOnlySubmission.stabilityScore],
              ['Motivasi', viewOnlySubmission.motivationScore],
            ] as [lbl, sc]}
              <div class="p-2.5 rounded-xl bg-slate-950 border border-slate-800 text-center">
                <div class="text-[10px] text-slate-400">{lbl}</div>
                <div class="text-base font-black text-emerald-400 mt-0.5">{sc} / 5</div>
              </div>
            {/each}
          </div>

          <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
            <div class="text-slate-400">Motivator Utama Belajar:</div>
            <div class="text-slate-200 mt-1">{viewOnlySubmission.primaryMotivator}</div>
          </div>

          {#if viewOnlySubmission.parentFeedbackNotes}
            <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
              <div class="text-slate-400">Pesan Khusus untuk Tim Follow-up Orang Tua:</div>
              <div class="text-slate-200 mt-1 whitespace-pre-wrap">{viewOnlySubmission.parentFeedbackNotes}</div>
            </div>
          {/if}
        </div>

        <div class="flex justify-end pt-2">
          <button onclick={() => (wizardOpen = false)} class="px-4 py-2 rounded-xl bg-slate-800 text-slate-200 text-xs font-semibold cursor-pointer">
            Tutup
          </button>
        </div>
      {:else}
        <!-- Interactive 9-Step Wizard -->
        <div class="flex items-start justify-between border-b border-slate-800 pb-3">
          <div>
            <div class="flex items-center gap-2">
              <span class="text-[10px] font-bold uppercase px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-300">
                Langkah {currentStep} dari 9
              </span>
              <span class="text-[11px] text-emerald-400">💾 Tersimpan otomatis di draf</span>
            </div>
            <h3 class="text-lg font-bold text-white mt-1">Evaluasi Siswa: {activeTarget.studentName}</h3>
            <div class="text-xs text-slate-400">{activeTarget.groupTitle} ({activeTarget.courseName})</div>
          </div>
          <button onclick={() => (wizardOpen = false)} class="px-2.5 py-1 rounded-lg bg-slate-800 text-xs text-slate-300 cursor-pointer">✕</button>
        </div>

        <!-- Progress Tracker Bar -->
        <div class="w-full bg-slate-800 h-1.5 rounded-full overflow-hidden">
          <div class="bg-indigo-500 h-full transition-all duration-300" style="width: {(currentStep / 9) * 100}%"></div>
        </div>

        <!-- Step Views -->
        <div class="min-h-56">
          {#if currentStep === 1}
            <div class="space-y-3">
              <h4 class="text-sm font-bold text-white">1. Nominasi Penghargaan Karakter & Prestasi Siswa</h4>
              <p class="text-xs text-slate-400">Pilih badge penghargaan yang paling merefleksikan pencapaian siswa selama mengikuti kursus:</p>
              <div class="space-y-2 mt-2">
                {#each badgesList as b}
                  <label class="flex items-center gap-3 p-3 rounded-xl border cursor-pointer transition {nominationBadge === b ? 'bg-indigo-600/15 border-indigo-500 text-white font-semibold' : 'bg-slate-950 border-slate-800 text-slate-300 hover:border-slate-700'}">
                    <input type="radio" bind:group={nominationBadge} value={b} class="rounded text-indigo-600" />
                    <span class="text-xs">{b}</span>
                  </label>
                {/each}
              </div>
            </div>
          {:else if currentStep === 2}
            <div class="space-y-3">
              <h4 class="text-sm font-bold text-white">2. Observasi & Perkembangan Siswa</h4>
              <p class="text-xs text-slate-400">Tuliskan narasi perkembangan pemahaman materi, antusiasme, dan kendala yang berhasil diatasi siswa:</p>
              <textarea
                bind:value={observationNotes}
                rows="6"
                placeholder="Contoh: Ananda sangat cepat memahami logika looping dan nested if. Pada project terakhir mampu membuat custom function sendiri tanpa banyak bantuan..."
                class="w-full p-3.5 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white focus:outline-none focus:border-indigo-500 leading-relaxed"
              ></textarea>
            </div>
          {:else if currentStep === 3}
            <div class="space-y-3">
              <h4 class="text-sm font-bold text-white">3. Skala Pemahaman Konsep Koding</h4>
              <p class="text-xs text-slate-400">Sejauh mana siswa memahami sintaks, algoritma logika, dan struktur kode (1 = Masih sangat butuh bantuan, 5 = Paham mendalam & mahir):</p>
              <div class="flex items-center justify-between gap-2 pt-3">
                {#each [1, 2, 3, 4, 5] as sc}
                  <button
                    type="button"
                    onclick={() => (comprehensionScore = sc)}
                    class="flex-1 py-3 rounded-xl border text-sm font-black transition cursor-pointer {comprehensionScore === sc ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'}"
                  >
                    {sc}
                  </button>
                {/each}
              </div>
            </div>
          {:else if currentStep === 4}
            <div class="space-y-3">
              <h4 class="text-sm font-bold text-white">4. Skala Kemandirian Mengerjakan Proyek</h4>
              <p class="text-xs text-slate-400">Kemampuan siswa dalam mengerjakan modul latihan mandiri dan eksplorasi fitur koding sendiri:</p>
              <div class="flex items-center justify-between gap-2 pt-3">
                {#each [1, 2, 3, 4, 5] as sc}
                  <button
                    type="button"
                    onclick={() => (independenceScore = sc)}
                    class="flex-1 py-3 rounded-xl border text-sm font-black transition cursor-pointer {independenceScore === sc ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'}"
                  >
                    {sc}
                  </button>
                {/each}
              </div>
            </div>
          {:else if currentStep === 5}
            <div class="space-y-3">
              <h4 class="text-sm font-bold text-white">5. Skala Perhatian & Fokus Belajar</h4>
              <p class="text-xs text-slate-400">Tingkat konsentrasi, ketertiban dalam sesi daring, dan daya tangkap saat tutor menjelaskan:</p>
              <div class="flex items-center justify-between gap-2 pt-3">
                {#each [1, 2, 3, 4, 5] as sc}
                  <button
                    type="button"
                    onclick={() => (focusScore = sc)}
                    class="flex-1 py-3 rounded-xl border text-sm font-black transition cursor-pointer {focusScore === sc ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'}"
                  >
                    {sc}
                  </button>
                {/each}
              </div>
            </div>
          {:else if currentStep === 6}
            <div class="space-y-3">
              <h4 class="text-sm font-bold text-white">6. Skala Stabilitas Emosional & Resiliensi Bug</h4>
              <p class="text-xs text-slate-400">Ketahanan siswa saat menghadapi error atau kodingan macet (apakah tenang mencari solusi atau mudah putus asa):</p>
              <div class="flex items-center justify-between gap-2 pt-3">
                {#each [1, 2, 3, 4, 5] as sc}
                  <button
                    type="button"
                    onclick={() => (stabilityScore = sc)}
                    class="flex-1 py-3 rounded-xl border text-sm font-black transition cursor-pointer {stabilityScore === sc ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'}"
                  >
                    {sc}
                  </button>
                {/each}
              </div>
            </div>
          {:else if currentStep === 7}
            <div class="space-y-3">
              <h4 class="text-sm font-bold text-white">7. Skala Motivasi Melanjutkan ke Modul Berikutnya</h4>
              <p class="text-xs text-slate-400">Antusiasme siswa untuk memperpanjang paket dan naik ke level kurikulum yang lebih menantang:</p>
              <div class="flex items-center justify-between gap-2 pt-3">
                {#each [1, 2, 3, 4, 5] as sc}
                  <button
                    type="button"
                    onclick={() => (motivationScore = sc)}
                    class="flex-1 py-3 rounded-xl border text-sm font-black transition cursor-pointer {motivationScore === sc ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'}"
                  >
                    {sc}
                  </button>
                {/each}
              </div>
            </div>
          {:else if currentStep === 8}
            <div class="space-y-3">
              <h4 class="text-sm font-bold text-white">8. Faktor Motivator Utama Siswa</h4>
              <p class="text-xs text-slate-400">Apa pemicu semangat terbesar bagi siswa dalam belajar koding? Klik salah satu rekomendasi atau tuliskan sendiri:</p>
              <input
                bind:value={primaryMotivator}
                placeholder="Tuliskan faktor motivasi utama siswa..."
                class="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white focus:outline-none focus:border-indigo-500"
              />
              <div class="space-y-1.5 pt-2">
                <div class="text-[11px] text-slate-400 font-semibold">Saran Cepat:</div>
                {#each motivatorSuggestions as s}
                  <button
                    type="button"
                    onclick={() => (primaryMotivator = s)}
                    class="block w-full text-left p-2 rounded-lg bg-slate-950 border border-slate-800/80 hover:border-indigo-500 text-[11px] text-slate-300 cursor-pointer transition"
                  >
                    💡 {s}
                  </button>
                {/each}
              </div>
            </div>
          {:else if currentStep === 9}
            <div class="space-y-3">
              <h4 class="text-sm font-bold text-white">9. Catatan Komunikasi & Rekomendasi untuk Orang Tua</h4>
              <p class="text-xs text-slate-400">Poin penting untuk disampaikan ke tim Sales/Retention saat konsultasi perpanjangan paket kursus dengan orang tua:</p>
              <textarea
                bind:value={parentFeedbackNotes}
                rows="5"
                placeholder="Contoh: Orang tua sangat mengapresiasi kemajuan putranya. Disarankan menawarkan jadwal tetap di Sabtu pagi untuk modul Python berikutnya..."
                class="w-full p-3.5 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white focus:outline-none focus:border-indigo-500 leading-relaxed"
              ></textarea>
            </div>
          {/if}
        </div>

        <!-- Wizard Navigation Footer -->
        <div class="flex items-center justify-between border-t border-slate-800 pt-3">
          <button
            type="button"
            onclick={prevStep}
            disabled={currentStep === 1}
            class="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed text-xs text-slate-300 font-semibold cursor-pointer transition"
          >
            ← Kembali
          </button>

          <div class="flex items-center gap-2">
            {#if currentStep < 9}
              <button
                type="button"
                onclick={nextStep}
                class="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer transition shadow-md shadow-indigo-600/30"
              >
                Selanjutnya →
              </button>
            {:else}
              <button
                type="button"
                onclick={submitSurvey}
                disabled={submitting}
                class="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold cursor-pointer transition shadow-md shadow-emerald-600/30"
              >
                {submitting ? 'Mengirim...' : '✓ Kirim Evaluasi Siswa'}
              </button>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

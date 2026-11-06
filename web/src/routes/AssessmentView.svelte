<script lang="ts">
  import { apiFetch, app, fmtDateTimeWIB, notify } from '../lib/state.svelte';

  let data = $state<any>(app.cache.assessments || null);
  let answers = $state<Record<string, any>>({});
  let programs = $state<string[]>(['coding']);
  let chartMode = $state<'cycle' | 'month'>('cycle');
  let submitting = $state(false);

  async function loadAssessments() {
    const res = await apiFetch('/api/v1/assessments');
    data = res;
    app.cache.assessments = res;

    if (res.activeCycle && !res.activeCycle.submission) {
      const draftKey = `nk.assessment.draft.${app.tutor?.userId || 'local'}.${res.activeCycle.cycleCode}`;
      const raw = localStorage.getItem(draftKey);
      if (raw) {
        try {
          const parsed = JSON.parse(raw);
          answers = parsed.values || {};
          programs = parsed.programs || ['coding'];
        } catch {
          // ignore
        }
      }
    }
  }

  function saveDraft() {
    if (!data?.activeCycle) return;
    const draftKey = `nk.assessment.draft.${app.tutor?.userId || 'local'}.${data.activeCycle.cycleCode}`;
    localStorage.setItem(draftKey, JSON.stringify({ savedAt: Date.now(), values: answers, programs }));
  }

  async function submitCycle(e: Event) {
    e.preventDefault();
    if (!data?.activeCycle) return;
    submitting = true;
    try {
      await apiFetch(`/api/v1/assessments/${data.activeCycle.cycleCode}/submit`, {
        method: 'POST',
        body: JSON.stringify({ programs, answers }),
      });
      localStorage.removeItem(`nk.assessment.draft.${app.tutor?.userId || 'local'}.${data.activeCycle.cycleCode}`);
      await loadAssessments();
      notify('Assessment berhasil dikumpulkan! Skor langsung tercatat di riwayat.', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    } finally {
      submitting = false;
    }
  }

  $effect(() => {
    loadAssessments();
  });

  const trendPoints = $derived.by(() => {
    const hist: any[] = (data?.history || []).slice(0, 12).reverse();
    if (hist.length === 0) return [];
    return hist.map((h, i) => ({
      label: chartMode === 'cycle' ? h.cycleCode.slice(-4) : h.monthKey.slice(5),
      score: h.scorePct,
      x: 35 + i * (520 / Math.max(hist.length - 1, 1)),
      y: 130 - ((h.scorePct - 70) / 30) * 100,
    }));
  });

  const polylineStr = $derived(trendPoints.map((p) => `${p.x},${p.y}`).join(' '));
</script>

<div class="space-y-5">
  <div>
    <h2 class="text-xl font-bold text-white">Tutor Assessment & Riwayat Kompetensi</h2>
    <p class="text-xs text-slate-400">Penyimpanan draf otomatis di browser & grafik riwayat kompetensi penilaian.</p>
  </div>

  <!-- Active Cycle Card -->
  {#if data?.activeCycle}
    <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <span class="text-[11px] font-bold px-2.5 py-0.5 rounded-full bg-indigo-600/20 text-indigo-300">
            Siklus Aktif: {data.activeCycle.cycleCode}
          </span>
          <h3 class="mt-1.5 text-base font-bold text-white">{data.activeCycle.title}</h3>
          <div class="text-xs text-slate-400">Tenggat Pengumpulan: {fmtDateTimeWIB(data.activeCycle.deadlineAt)}</div>
        </div>

        {#if data.activeCycle.submission}
          <div class="px-4 py-2 rounded-xl bg-emerald-500/15 border border-emerald-500/30 text-xs font-bold text-emerald-300">
            ✓ Sudah Dikumpulkan ({fmtDateTimeWIB(data.activeCycle.submission.submittedAt)}) • Skor: {data.activeCycle.submission.scorePct}%
          </div>
        {/if}
      </div>

      {#if !data.activeCycle.submission}
        <form onsubmit={submitCycle} class="space-y-4 pt-3 border-t border-slate-800">
          {#each data.activeCycle.questions || [] as q, idx}
            <div class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2.5 text-xs">
              <div class="text-[11px] font-semibold text-indigo-400">{q.section}</div>
              <div class="font-bold text-white text-sm">{idx + 1}. {q.prompt}</div>

              {#if q.kind === 'single'}
                <div class="space-y-1.5">
                  {#each q.options || [] as opt}
                    <label class="flex items-start gap-2.5 p-2 rounded-lg hover:bg-slate-900 cursor-pointer">
                      <input
                        type="radio"
                        name={q.id}
                        value={opt}
                        checked={answers[q.id] === opt}
                        onchange={() => {
                          answers[q.id] = opt;
                          saveDraft();
                        }}
                        required={q.required}
                        class="mt-0.5"
                      />
                      <span class="text-slate-300">{opt}</span>
                    </label>
                  {/each}
                </div>
              {:else if q.kind === 'scale'}
                <div class="flex items-center gap-2">
                  {#each [1, 2, 3, 4, 5] as n}
                    <button
                      type="button"
                      onclick={() => {
                        answers[q.id] = n;
                        saveDraft();
                      }}
                      class="w-10 h-10 rounded-xl font-bold border cursor-pointer {answers[q.id] === n ? 'bg-indigo-600 border-indigo-400 text-white' : 'bg-slate-900 border-slate-800 text-slate-400'}"
                    >
                      {n}
                    </button>
                  {/each}
                </div>
              {:else}
                <textarea
                  rows="3"
                  required={q.required}
                  value={answers[q.id] || ''}
                  oninput={(e) => {
                    answers[q.id] = (e.target as HTMLTextAreaElement).value;
                    saveDraft();
                  }}
                  placeholder="Tuliskan jawaban studi kasus Anda..."
                  class="w-full p-3 rounded-xl bg-slate-900 border border-slate-800 text-white"
                ></textarea>
              {/if}
            </div>
          {/each}

          <button type="submit" disabled={submitting} class="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer">
            {submitting ? 'Mengirim...' : 'Kumpulkan Assessment Siklus Ini ⚡'}
          </button>
        </form>
      {/if}
    </div>
  {/if}

  <!-- History & SVG Trend Chart -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
    <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-3">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-bold text-white">Tren Nilai Assessment (12 Siklus Terakhir)</h3>
        <div class="flex gap-1 text-xs">
          <button onclick={() => (chartMode = 'cycle')} class="px-2.5 py-1 rounded-lg cursor-pointer {chartMode === 'cycle' ? 'bg-indigo-600 text-white' : 'bg-slate-800 text-slate-400'}">Per Siklus</button>
          <button onclick={() => (chartMode = 'month')} class="px-2.5 py-1 rounded-lg cursor-pointer {chartMode === 'month' ? 'bg-indigo-600 text-white' : 'bg-slate-800 text-slate-400'}">Per Bulan</button>
        </div>
      </div>

      <svg viewBox="0 0 580 160" class="w-full h-44 bg-slate-950 rounded-xl border border-slate-800 p-2">
        <line x1="30" y1="30" x2="560" y2="30" stroke="#1e293b" stroke-dasharray="4" />
        <line x1="30" y1="80" x2="560" y2="80" stroke="#1e293b" stroke-dasharray="4" />
        <line x1="30" y1="130" x2="560" y2="130" stroke="#1e293b" />
        {#if polylineStr}
          <polyline fill="none" stroke="#6366f1" stroke-width="3" points={polylineStr} />
          {#each trendPoints as p}
            <circle cx={p.x} cy={p.y} r="4.5" fill="#10b981" />
            <text x={p.x} y="150" text-anchor="middle" fill="#64748b" font-size="10">{p.label}</text>
            <text x={p.x} y={p.y - 8} text-anchor="middle" fill="#e2e8f0" font-size="10" font-weight="bold">{p.score}%</text>
          {/each}
        {/if}
      </svg>
    </div>

    <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-3">
      <h3 class="text-sm font-bold text-white">Riwayat Nilai Siklus ({data?.history?.length ?? 0} Siklus)</h3>
      <div class="max-h-60 overflow-y-auto divide-y divide-slate-800/70 text-xs">
        {#each data?.history || [] as h}
          <div class="py-2.5 flex items-center justify-between">
            <div>
              <div class="font-semibold text-white">{h.cycleLabel}</div>
              <div class="text-[11px] text-slate-500">{h.program} • {h.monthKey}</div>
            </div>
            <span class="px-2.5 py-0.5 rounded-full font-bold bg-emerald-500/15 text-emerald-400">{h.scorePct}%</span>
          </div>
        {/each}
      </div>
    </div>
  </div>
</div>

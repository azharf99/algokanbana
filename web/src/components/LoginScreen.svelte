<script lang="ts">
  import { apiFetch, connectSSE, notify, refreshBootstrap } from '../lib/state.svelte';

  let login = $state('azharfa');
  let password = $state('Azhar1995!');
  let verifyLms = $state(false);
  let loading = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    loading = true;
    try {
      await apiFetch('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ login, password, verifyLms }),
      });
      await refreshBootstrap();
      connectSSE();
      notify('Berhasil masuk ke AlgoKanbana!', 'success');
    } catch (err: any) {
      notify(err.message || 'Gagal login', 'error');
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen bg-slate-950 flex items-center justify-center p-4">
  <div class="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-7 shadow-2xl">
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-xl bg-indigo-600 flex items-center justify-center font-black text-lg text-white shadow-lg shadow-indigo-600/30">
          AK
        </div>
        <div>
          <h1 class="text-lg font-bold text-white tracking-tight">AlgoKanbana</h1>
          <p class="text-xs text-slate-400">Algonova Indonesia • Ultra-Fast Tutor Portal</p>
        </div>
      </div>
      <span class="text-[11px] font-semibold px-2.5 py-1 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
        Go Single-Binary
      </span>
    </div>

    <div class="mb-5 p-3.5 rounded-xl bg-slate-950/80 border border-slate-800 text-xs text-slate-300 space-y-1">
      <div class="font-semibold text-indigo-400">Mode Aman Lokal Aktif (Zero Critical POST)</div>
      <p class="text-slate-400 leading-relaxed">
        Seluruh perubahan data disimpan di SQLite WAL lokal (<code class="text-emerald-400">&lt; 1 ms</code>) tanpa menyentuh endpoint mutasi produksi Algonova.
      </p>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div>
        <label for="ak-login" class="block text-xs font-medium text-slate-300 mb-1.5">Username Akun LMS</label>
        <input
          id="ak-login"
          bind:value={login}
          required
          class="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-sm text-white focus:outline-none focus:border-indigo-500"
          placeholder="azharfa"
        />
      </div>

      <div>
        <label for="ak-pass" class="block text-xs font-medium text-slate-300 mb-1.5">Password LMS</label>
        <input
          id="ak-pass"
          type="password"
          bind:value={password}
          required
          class="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-sm text-white focus:outline-none focus:border-indigo-500"
        />
      </div>

      <label class="flex items-start gap-2.5 text-xs text-slate-400 cursor-pointer select-none pt-1">
        <input type="checkbox" bind:checked={verifyLms} class="mt-0.5 rounded border-slate-700 bg-slate-950 text-indigo-600" />
        <span>Verifikasi kredensial langsung ke <code class="text-slate-300">lms.algonova.id</code> (Read-Only Auth)</span>
      </label>

      <button
        type="submit"
        disabled={loading}
        class="w-full py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold text-sm transition cursor-pointer shadow-lg shadow-indigo-600/25"
      >
        {loading ? 'Memproses Sesi...' : 'Masuk Instan ke Portal Tutor ⚡'}
      </button>
    </form>

    <div class="mt-6 pt-4 border-t border-slate-800/80 grid grid-cols-3 gap-2 text-center text-[11px]">
      <div class="p-2 rounded-lg bg-slate-950/60">
        <div class="text-emerald-400 font-bold">&lt; 5 ms</div>
        <div class="text-slate-500">Respon API</div>
      </div>
      <div class="p-2 rounded-lg bg-slate-950/60">
        <div class="text-indigo-400 font-bold">~18 MB</div>
        <div class="text-slate-500">Docker Image</div>
      </div>
      <div class="p-2 rounded-lg bg-slate-950/60">
        <div class="text-amber-400 font-bold">+03 &rarr; WIB</div>
        <div class="text-slate-500">Auto Timezone</div>
      </div>
    </div>
  </div>
</div>

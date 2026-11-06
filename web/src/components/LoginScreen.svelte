<script lang="ts">
  import { apiFetch, connectSSE, notify, refreshBootstrap } from '../lib/state.svelte';

  let login = $state('');
  let password = $state('');
  let verifyLms = $state(true);
  let loading = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!login.trim() || !password.trim()) {
      notify('Username dan password LMS wajib diisi', 'error');
      return;
    }
    loading = true;
    try {
      await apiFetch('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ login: login.trim(), password, verifyLms }),
      });
      await refreshBootstrap();
      connectSSE();
      notify('Berhasil masuk ke AlgoKanbana!', 'success');
    } catch (err: any) {
      notify(err.message || 'Gagal login, periksa kembali username dan password LMS', 'error');
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
          <p class="text-xs text-slate-400">Portal Pengajaran Tutor • Algonova Indonesia</p>
        </div>
      </div>
      <span class="text-[11px] font-semibold px-2.5 py-1 rounded-full bg-indigo-500/15 text-indigo-300 border border-indigo-500/30">
        Tutor Portal
      </span>
    </div>

    <div class="mb-5 p-3.5 rounded-xl bg-slate-950/80 border border-slate-800 text-xs text-slate-300 space-y-1">
      <div class="font-semibold text-slate-200">Autentikasi Akun LMS Algonova</div>
      <p class="text-slate-400 leading-relaxed">
        Gunakan username dan password yang sama dengan akun <code class="text-indigo-300">lms.algonova.id</code> Anda untuk mengakses seluruh jadwal, siswa, dan tugas.
      </p>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div>
        <label for="ak-login" class="block text-xs font-medium text-slate-300 mb-1.5">Username LMS</label>
        <input
          id="ak-login"
          bind:value={login}
          required
          class="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-sm text-white focus:outline-none focus:border-indigo-500 placeholder-slate-600"
          placeholder="Masukkan username LMS Anda..."
        />
      </div>

      <div>
        <label for="ak-pass" class="block text-xs font-medium text-slate-300 mb-1.5">Password LMS</label>
        <input
          id="ak-pass"
          type="password"
          bind:value={password}
          required
          class="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-sm text-white focus:outline-none focus:border-indigo-500 placeholder-slate-600"
          placeholder="Masukkan password akun LMS..."
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        class="w-full py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold text-sm transition cursor-pointer shadow-lg shadow-indigo-600/25"
      >
        {loading ? 'Memverifikasi Akun...' : 'Masuk ke Portal Tutor'}
      </button>
    </form>

    <div class="mt-6 pt-4 border-t border-slate-800/80 text-center text-[11px] text-slate-500">
      Algonova Indonesia &bull; High-Performance Tutor Daily Driver
    </div>
  </div>
</div>

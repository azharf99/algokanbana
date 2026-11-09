// Global reactive application store & ultra-fast API client (Svelte 5 runes)

export type RouteKey =
  | 'dashboard'
  | 'taskboard'
  | 'students'
  | 'regular-feedback'
  | 'schedule'
  | 'assessment'
  | 'surveys'
  | 'communication'
  | 'profile'
  | 'tl-demo'
  | 'benchmark';

export interface ToastItem {
  id: number;
  message: string;
  kind: 'success' | 'error' | 'info';
}

export const app = $state({
  ready: false,
  authenticated: false,
  route: 'dashboard' as RouteKey,
  routeParam: '' as string,
  paletteOpen: false,
  session: null as any,
  tutor: null as any,
  upcomingLessons: [] as any[],
  savedViews: [] as any[],
  portalEvents: [] as any[],
  badges: { tutorUnread: 0, tlUnread: 0, assessmentOpen: false },
  sync: null as any,
  telemetry: {
    serverMs: 0.4,
    roundtripMs: 1.2,
    lastEndpoint: '/api/v1/bootstrap',
  },
  cache: {} as Record<string, any>,
  toasts: [] as ToastItem[],
  sseConnected: false,
});

let toastSeq = 1;

export function notify(message: string, kind: 'success' | 'error' | 'info' = 'success') {
  const id = toastSeq++;
  app.toasts.push({ id, message, kind });
  setTimeout(() => {
    const idx = app.toasts.findIndex((t) => t.id === id);
    if (idx !== -1) app.toasts.splice(idx, 1);
  }, 3600);
}

export function navigate(route: RouteKey, param = '') {
  const t0 = performance.now();
  app.route = route;
  app.routeParam = param;
  const url = `/portal/${route}${param ? `?ref=${encodeURIComponent(param)}` : ''}`;
  if (window.location.pathname + window.location.search !== url) {
    window.history.pushState({}, '', url);
  }
  app.telemetry.roundtripMs = Number(Math.max(0.1, performance.now() - t0).toFixed(2));
}

export async function apiFetch<T = any>(path: string, init?: RequestInit): Promise<T> {
  const t0 = performance.now();
  const headers = new Headers(init?.headers || {});
  if (init?.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  const resp = await fetch(path, {
    credentials: 'same-origin',
    ...init,
    headers,
  });
  const rt = performance.now() - t0;
  const srvHdr = resp.headers.get('X-Response-Time-Ms');
  app.telemetry.serverMs = srvHdr ? parseFloat(srvHdr) : Number((rt * 0.4).toFixed(2));
  app.telemetry.roundtripMs = Number(rt.toFixed(2));
  app.telemetry.lastEndpoint = path;

  const data = await resp.json().catch(() => ({}));
  if (!resp.ok) {
    if (resp.status === 401 && !path.includes('/auth/')) {
      app.authenticated = false;
    }
    throw new Error(data?.error || `HTTP ${resp.status}`);
  }
  return data as T;
}

export async function uploadFiles(files: FileList | File[], kind = 'proof'): Promise<string[]> {
  const fd = new FormData();
  fd.append('kind', kind);
  Array.from(files).forEach((f) => fd.append('files', f));
  const res = await apiFetch<{ urls: string[] }>('/api/v1/uploads', {
    method: 'POST',
    body: fd,
  });
  return res.urls || [];
}

export async function refreshBootstrap() {
  const b = await apiFetch('/api/v1/bootstrap');
  app.authenticated = true;
  app.session = b.session;
  app.tutor = b.tutor;
  app.upcomingLessons = b.upcomingLessons || [];
  app.savedViews = b.savedViews || [];
  app.portalEvents = b.portalEvents || [];
  app.badges = b.badges || app.badges;
  app.sync = b.sync;
}

let sseInstance: EventSource | null = null;
export function connectSSE() {
  if (sseInstance) sseInstance.close();
  sseInstance = new EventSource('/api/v1/events/stream');
  sseInstance.onopen = () => {
    app.sseConnected = true;
  };
  sseInstance.onerror = () => {
    app.sseConnected = false;
  };
  sseInstance.onmessage = (ev) => {
    try {
      const payload = JSON.parse(ev.data);
      if (payload.event && payload.event !== 'connected') {
        app.cache = {};
        refreshBootstrap().catch(() => {});
        window.dispatchEvent(new CustomEvent('ak-sse', { detail: payload }));
      }
    } catch {
      // ignore malformed keepalive
    }
  };
}

export function disconnectSSE() {
  if (sseInstance) {
    sseInstance.close();
    sseInstance = null;
  }
  app.sseConnected = false;
}

export function fmtDateTimeWIB(iso: string): string {
  if (!iso) return '-';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return (
    d.toLocaleDateString('id-ID', {
      weekday: 'short',
      day: '2-digit',
      month: 'short',
      timeZone: 'Asia/Jakarta',
    }) +
    ' • ' +
    d.toLocaleTimeString('id-ID', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
      timeZone: 'Asia/Jakarta',
    }) +
    ' WIB'
  );
}

export function fmtTimeOnlyWIB(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  return d.toLocaleTimeString('id-ID', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Jakarta',
  });
}

export async function copyToClipboard(text: string, label = 'Teks') {
  await navigator.clipboard.writeText(text);
  notify(`${label} berhasil disalin!`, 'info');
}

export function buildWhatsAppURL(phone: string, text: string): string {
  const clean = (phone || '').replace(/[^0-9]/g, '');
  const normalized = clean.startsWith('0') ? '62' + clean.slice(1) : clean;
  return `https://api.whatsapp.com/send?phone=${normalized}&text=${encodeURIComponent(text)}`;
}

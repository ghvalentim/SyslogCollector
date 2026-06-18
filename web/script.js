lucide.createIcons();

let isPaused = false;
function toggleAutoRefresh() {
    const tbody = document.getElementById('tabela-logs');
    if(!tbody) return; 
    
    const btnText = document.getElementById('text-refresh');
    const btnIcon = document.getElementById('icon-refresh');
    const btn = document.getElementById('btn-refresh');

    if (isPaused) {
        tbody.setAttribute('hx-trigger', 'every 2s');
        htmx.process(tbody); 
        btnText.innerText = "Pausar Live";
        btn.classList.replace('bg-slate-100', 'bg-blue-50');
        btn.classList.replace('text-slate-700', 'text-blue-700');
        btn.classList.replace('border-slate-300', 'border-blue-200');
        btnIcon.setAttribute('data-lucide', 'pause-circle');
    } else {
        tbody.setAttribute('hx-trigger', 'none');
        htmx.process(tbody);
        btnText.innerText = "Retomar Live";
        btn.classList.replace('bg-blue-50', 'bg-slate-100');
        btn.classList.replace('text-blue-700', 'text-slate-700');
        btn.classList.replace('border-blue-200', 'border-slate-300');
        btnIcon.setAttribute('data-lucide', 'play-circle');
    }
    lucide.createIcons();
    isPaused = !isPaused;
}

// Lógica de exportação agora apanha TODOS os filtros do formulário
function exportarCSV() {
    const form = document.getElementById('filter-form');
    if(form) {
        const query = new URLSearchParams(new FormData(form)).toString();
        window.location.href = `/export?${query}`;
    } else {
        window.location.href = '/export';
    }
}

function openLogDetails(btn) {
    if (!isPaused) toggleAutoRefresh();

    document.getElementById('detail-id').innerText = '#' + btn.getAttribute('data-id');
    document.getElementById('detail-ts').innerText = btn.getAttribute('data-ts');
    document.getElementById('detail-ip').innerText = btn.getAttribute('data-ip');
    document.getElementById('detail-proto').innerText = btn.getAttribute('data-proto');
    document.getElementById('detail-host').innerText = btn.getAttribute('data-host');
    document.getElementById('detail-app').innerText = btn.getAttribute('data-app');
    document.getElementById('detail-fac').innerText = btn.getAttribute('data-fac');
    
    const sev = btn.getAttribute('data-sev');
    const sevEl = document.getElementById('detail-sev');
    sevEl.innerText = sev;
    sevEl.className = 'px-2.5 py-1 text-xs uppercase tracking-wider font-bold rounded-md border inline-block';
    
    if (sev === 'Emergência') sevEl.classList.add('bg-red-100', 'text-red-800', 'border-red-200');
    else if (sev === 'Alerta') sevEl.classList.add('bg-orange-100', 'text-orange-800', 'border-orange-200');
    else if (sev === 'Crítico') sevEl.classList.add('bg-red-50', 'text-red-700', 'border-red-200');
    else if (sev === 'Erro') sevEl.classList.add('bg-red-50', 'text-red-600', 'border-red-100');
    else if (sev === 'Aviso') sevEl.classList.add('bg-yellow-50', 'text-yellow-700', 'border-yellow-200');
    else if (sev === 'Notice') sevEl.classList.add('bg-blue-50', 'text-blue-700', 'border-blue-200');
    else if (sev === 'Debug') sevEl.classList.add('bg-slate-100', 'text-slate-600', 'border-slate-200');
    else sevEl.classList.add('bg-emerald-50', 'text-emerald-700', 'border-emerald-200'); 

    document.getElementById('detail-payload').innerText = btn.getAttribute('data-payload');

    const backdrop = document.getElementById('drawer-backdrop');
    const drawer = document.getElementById('log-drawer');
    backdrop.classList.remove('hidden');
    setTimeout(() => backdrop.classList.remove('opacity-0'), 10);
    drawer.classList.remove('translate-x-full');
}

function closeLogDetails() {
    const backdrop = document.getElementById('drawer-backdrop');
    const drawer = document.getElementById('log-drawer');
    drawer.classList.add('translate-x-full');
    backdrop.classList.add('opacity-0');
    setTimeout(() => backdrop.classList.add('hidden'), 300);
}

function copiarPayload() {
    const payload = document.getElementById('detail-payload').innerText;
    const textarea = document.createElement('textarea');
    textarea.value = payload;
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
}

// Se a sessão expirar nos pedidos de fundo do HTMX, redireciona suavemente
document.body.addEventListener('htmx:responseError', function(evt) {
    if(evt.detail.xhr.status === 401) window.location.href = '/login';
});
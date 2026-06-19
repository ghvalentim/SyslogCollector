// web/script.js
lucide.createIcons();

let isPaused = false;
function toggleAutoRefresh() {
    const tbody = document.getElementById('tabela-logs');
    if (!tbody) return; 
    
    const btnText = document.getElementById('text-refresh');
    const btnIcon = document.getElementById('icon-refresh');
    const btn = document.getElementById('btn-refresh');

    if (isPaused) {
        tbody.setAttribute('hx-trigger', 'every 2s');
        htmx.process(tbody); 
        btnText.innerText = "Pausar Live";
        btn.classList.replace('btn-emerald', 'btn-blue'); 
        btnIcon.setAttribute('data-lucide', 'pause-circle');
    } else {
        tbody.setAttribute('hx-trigger', 'none');
        htmx.process(tbody);
        btnText.innerText = "Retomar Live";
        btn.classList.replace('btn-blue', 'btn-emerald');
        btnIcon.setAttribute('data-lucide', 'play-circle');
    }
    lucide.createIcons();
    isPaused = !isPaused;
}

function exportarCSV() {
    const form = document.getElementById('filter-form');
    const query = form ? new URLSearchParams(new FormData(form)).toString() : '';
    window.location.href = `/export?${query}`;
}

function exportarPDF() {
    const form = document.getElementById('filter-form');
    const query = form ? new URLSearchParams(new FormData(form)).toString() : '';
    window.location.href = `/export/pdf?${query}`;
}

function openLogDetails(btn) {
    if (!isPaused) toggleAutoRefresh();

    document.getElementById('detail-id').innerText = '#' + btn.getAttribute('data-id');
    document.getElementById('detail-ts').innerText = btn.getAttribute('data-ts');
    document.getElementById('detail-ip').innerText = btn.getAttribute('data-ip');
    document.getElementById('detail-proto').innerText = btn.getAttribute('data-proto');
    document.getElementById('detail-host').innerText = btn.getAttribute('data-host');
    document.getElementById('detail-app').innerText = btn.getAttribute('data-app');
    
    // Novas Colunas: SourceType e Facility Humanizada
    document.getElementById('detail-source').innerText = btn.getAttribute('data-source') || 'Unknown';
    document.getElementById('detail-facname').innerText = 'Facility: ' + (btn.getAttribute('data-facname') || '-');
    document.getElementById('detail-fac').innerText = btn.getAttribute('data-fac'); // Legado
    
    const sev = btn.getAttribute('data-sev');
    const sevEl = document.getElementById('detail-sev');
    sevEl.innerText = sev;
    sevEl.className = 'badge';
    
    if (sev === 'Emergência') sevEl.classList.add('badge-emergencia');
    else if (sev === 'Alerta') sevEl.classList.add('badge-alerta');
    else if (sev === 'Crítico') sevEl.classList.add('badge-critico');
    else if (sev === 'Erro') sevEl.classList.add('badge-erro');
    else if (sev === 'Aviso') sevEl.classList.add('badge-aviso');
    else if (sev === 'Notice') sevEl.classList.add('badge-notice');
    else if (sev === 'Debug') sevEl.classList.add('badge-debug');
    else sevEl.classList.add('badge-info'); 

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

document.body.addEventListener('htmx:responseError', function(evt) {
    if(evt.detail.xhr.status === 401) window.location.href = '/login';
});
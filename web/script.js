// Inicializar os ícones do Lucide
lucide.createIcons();

let isPaused = false;

// Função para Pausar/Retomar a tabela de logs
function toggleAutoRefresh() {
    const tbody = document.getElementById('tabela-logs');
    if (!tbody) return; // Segurança para não dar erro noutras vistas (Estatísticas, Ferramentas)
    
    const btnText = document.getElementById('text-refresh');
    const btnIcon = document.getElementById('icon-refresh');
    const btn = document.getElementById('btn-refresh');

    if (isPaused) {
        // Retomar
        tbody.setAttribute('hx-trigger', 'every 2s');
        htmx.process(tbody); 
        btnText.innerText = "Pausar Live";
        
        // Trocar as classes encapsuladas (Ver styles.css)
        btn.classList.replace('btn-emerald', 'btn-blue'); 
        btnIcon.setAttribute('data-lucide', 'pause-circle');
    } else {
        // Pausar
        tbody.setAttribute('hx-trigger', 'none');
        htmx.process(tbody);
        btnText.innerText = "Retomar Live";
        
        // Trocar as classes encapsuladas
        btn.classList.replace('btn-blue', 'btn-emerald');
        btnIcon.setAttribute('data-lucide', 'play-circle');
    }
    lucide.createIcons();
    isPaused = !isPaused;
}

// Lógica de exportação: Apanha os filtros de Texto e Gravidade
function exportarCSV() {
    const form = document.getElementById('filter-form');
    if (form) {
        // Pega em todos os dados do form (input 'q' e select 'sev') e constrói o URL
        const query = new URLSearchParams(new FormData(form)).toString();
        window.location.href = `/export?${query}`;
    } else {
        window.location.href = '/export';
    }
}

// --- LÓGICA DA JANELA LATERAL (DRAWER) ---
function openLogDetails(btn) {
    // Pausa a tabela automaticamente para o utilizador ler tranquilamente
    if (!isPaused) toggleAutoRefresh();

    // Preencher os dados usando os atributos injetados pelo backend
    document.getElementById('detail-id').innerText = '#' + btn.getAttribute('data-id');
    document.getElementById('detail-ts').innerText = btn.getAttribute('data-ts');
    document.getElementById('detail-ip').innerText = btn.getAttribute('data-ip');
    document.getElementById('detail-proto').innerText = btn.getAttribute('data-proto');
    document.getElementById('detail-host').innerText = btn.getAttribute('data-host');
    document.getElementById('detail-app').innerText = btn.getAttribute('data-app');
    document.getElementById('detail-fac').innerText = btn.getAttribute('data-fac');
    
    // Adicionar a classe correta ao Badge de gravidade (usando as classes do styles.css)
    const sev = btn.getAttribute('data-sev');
    const sevEl = document.getElementById('detail-sev');
    sevEl.innerText = sev;
    
    // Reset da classe base
    sevEl.className = 'badge';
    
    // Adicionar a cor
    if (sev === 'Emergência') sevEl.classList.add('badge-emergencia');
    else if (sev === 'Alerta') sevEl.classList.add('badge-alerta');
    else if (sev === 'Crítico') sevEl.classList.add('badge-critico');
    else if (sev === 'Erro') sevEl.classList.add('badge-erro');
    else if (sev === 'Aviso') sevEl.classList.add('badge-aviso');
    else if (sev === 'Notice') sevEl.classList.add('badge-notice');
    else if (sev === 'Debug') sevEl.classList.add('badge-debug');
    else sevEl.classList.add('badge-info'); 

    // Preencher o Payload completo no Terminal Escuro
    document.getElementById('detail-payload').innerText = btn.getAttribute('data-payload');

    // Mostrar a Janela Lateral (Drawer) com animação
    const backdrop = document.getElementById('drawer-backdrop');
    const drawer = document.getElementById('log-drawer');
    backdrop.classList.remove('hidden');
    setTimeout(() => backdrop.classList.remove('opacity-0'), 10);
    drawer.classList.remove('translate-x-full');
}

function closeLogDetails() {
    const backdrop = document.getElementById('drawer-backdrop');
    const drawer = document.getElementById('log-drawer');
    
    // Ocultar a janela suavemente
    drawer.classList.add('translate-x-full');
    backdrop.classList.add('opacity-0');
    setTimeout(() => backdrop.classList.add('hidden'), 300);
}

// Copiar mensagem rápida
function copiarPayload() {
    const payload = document.getElementById('detail-payload').innerText;
    const textarea = document.createElement('textarea');
    textarea.value = payload;
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
    
    alert("Mensagem copiada para a área de transferência!");
}

// Se a sessão expirar nos pedidos de fundo do HTMX, redireciona suavemente para o Login
document.body.addEventListener('htmx:responseError', function(evt) {
    if(evt.detail.xhr.status === 401) {
        window.location.href = '/login';
    }
});
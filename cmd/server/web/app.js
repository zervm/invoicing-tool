const loginSection = document.getElementById('login-section');
const appSection = document.getElementById('app-section');
const invoiceDetail = document.getElementById('invoice-detail');
const loginError = document.getElementById('login-error');

// Credentials held only in memory for this page load — never persisted
// to localStorage/sessionStorage, same reasoning as the booking system's
// admin page: a small re-login inconvenience is worth not storing
// credentials in the browser at all.
let authHeader = null;
let clientsCache = [];

function authedFetch(url, options = {}) {
  return fetch(url, {
    ...options,
    headers: { ...(options.headers || {}), Authorization: authHeader },
  });
}

function money(cents) {
  return '$' + (cents / 100).toFixed(2);
}

// ---------- Login ----------

document.getElementById('login-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const user = document.getElementById('admin-user').value;
  const pass = document.getElementById('admin-pass').value;
  authHeader = 'Basic ' + btoa(`${user}:${pass}`);

  const res = await authedFetch('/api/clients');
  if (res.status === 401) {
    loginError.textContent = 'Invalid credentials.';
    loginError.hidden = false;
    authHeader = null;
    return;
  }

  loginSection.hidden = true;
  appSection.hidden = false;
  loadEverything();
});

async function loadEverything() {
  await loadClients();
  await loadInvoices();
}

// ---------- Clients ----------

async function loadClients() {
  const res = await authedFetch('/api/clients');
  clientsCache = await res.json();

  document.querySelector('#clients-table tbody').innerHTML = clientsCache
    .map(c => `<tr><td>${c.name}</td><td>${c.email || ''}</td><td>${c.address || ''}</td></tr>`)
    .join('') || '<tr><td colspan="3">No clients yet.</td></tr>';

  const select = document.getElementById('invoice-client');
  select.innerHTML = clientsCache.map(c => `<option value="${c.id}">${c.name}</option>`).join('');
}

document.getElementById('client-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const messageEl = document.getElementById('client-message');

  const payload = {
    name: document.getElementById('client-name').value,
    email: document.getElementById('client-email').value,
    address: document.getElementById('client-address').value,
  };

  const res = await authedFetch('/api/clients', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const err = await res.json();
    messageEl.textContent = err.error || 'Could not create client.';
    messageEl.className = 'message error';
    messageEl.hidden = false;
    return;
  }

  document.getElementById('client-form').reset();
  messageEl.hidden = true;
  loadClients();
});

// ---------- Line items (dynamic rows) ----------

function addLineItemRow() {
  const container = document.getElementById('line-items');
  const row = document.createElement('div');
  row.className = 'line-item-row';
  row.innerHTML = `
    <input type="text" class="desc" placeholder="Description" required>
    <input type="number" class="qty" placeholder="Qty" value="1" min="0" step="0.25" required>
    <input type="number" class="price" placeholder="Unit price ($)" min="0" step="0.01" required>
    <button type="button" class="danger-btn remove-line">Remove</button>
  `;
  row.querySelector('.remove-line').addEventListener('click', () => row.remove());
  container.appendChild(row);
}
document.getElementById('add-line-item').addEventListener('click', addLineItemRow);
addLineItemRow(); // start with one row so the form isn't empty

// ---------- Invoices ----------

async function loadInvoices() {
  const res = await authedFetch('/api/invoices');
  const invoices = await res.json();

  document.querySelector('#invoices-table tbody').innerHTML = invoices
    .map(inv => `
      <tr>
        <td>${inv.number}</td>
        <td>${inv.client.name}</td>
        <td>${inv.due_date}</td>
        <td>${money(inv.total_cents)}</td>
        <td><span class="status-tag status-${inv.status}">${inv.status}</span></td>
        <td><button class="secondary-btn view-invoice" data-id="${inv.id}">View</button></td>
      </tr>
    `)
    .join('') || '<tr><td colspan="6">No invoices yet.</td></tr>';

  document.querySelectorAll('.view-invoice').forEach(btn => {
    btn.addEventListener('click', () => viewInvoice(btn.dataset.id));
  });
}

document.getElementById('invoice-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const messageEl = document.getElementById('invoice-message');

  const lineItems = Array.from(document.querySelectorAll('.line-item-row')).map(row => ({
    description: row.querySelector('.desc').value,
    quantity: parseFloat(row.querySelector('.qty').value) || 0,
    unit_price_cents: Math.round(parseFloat(row.querySelector('.price').value || 0) * 100),
  }));

  const payload = {
    client_id: document.getElementById('invoice-client').value,
    issue_date: document.getElementById('invoice-issue-date').value,
    due_date: document.getElementById('invoice-due-date').value,
    line_items: lineItems,
    tax_rate_percent: parseFloat(document.getElementById('invoice-tax').value) || 0,
    notes: document.getElementById('invoice-notes').value,
  };

  const res = await authedFetch('/api/invoices', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const err = await res.json();
    messageEl.textContent = err.error || 'Could not create invoice.';
    messageEl.className = 'message error';
    messageEl.hidden = false;
    return;
  }

  document.getElementById('invoice-form').reset();
  document.getElementById('line-items').innerHTML = '';
  addLineItemRow();
  messageEl.hidden = true;
  loadInvoices();
});

// ---------- Single invoice / print view ----------

let currentInvoiceId = null;

async function viewInvoice(id) {
  const res = await authedFetch(`/api/invoices/${id}`);
  if (!res.ok) return;
  const inv = await res.json();
  currentInvoiceId = id;
  renderInvoiceDetail(inv);

  appSection.hidden = true;
  invoiceDetail.hidden = false;
}

function renderInvoiceDetail(inv) {
  document.getElementById('inv-number').textContent = inv.number;
  document.getElementById('inv-status-tag').innerHTML =
    `<span class="status-tag status-${inv.status}">${inv.status}</span>`;
  document.getElementById('inv-issue-date').textContent = inv.issue_date;
  document.getElementById('inv-due-date').textContent = inv.due_date;
  document.getElementById('inv-client-name').textContent = inv.client.name;
  document.getElementById('inv-client-email').textContent = inv.client.email || '';
  document.getElementById('inv-client-address').textContent = inv.client.address || '';

  document.getElementById('inv-line-items').innerHTML = inv.line_items
    .map(li => `<tr>
      <td>${li.description}</td>
      <td>${li.quantity}</td>
      <td>${money(li.unit_price_cents)}</td>
      <td>${money(li.line_total_cents)}</td>
    </tr>`)
    .join('');

  document.getElementById('inv-subtotal').textContent = money(inv.subtotal_cents);
  document.getElementById('inv-tax').textContent = money(inv.tax_cents) + ` (${inv.tax_rate_percent}%)`;
  document.getElementById('inv-total').textContent = money(inv.total_cents);
  document.getElementById('inv-notes').textContent = inv.notes || '';
}

document.getElementById('back-to-list').addEventListener('click', (e) => {
  e.preventDefault();
  invoiceDetail.hidden = true;
  appSection.hidden = false;
});

document.getElementById('print-btn').addEventListener('click', () => window.print());

async function updateStatus(status) {
  await authedFetch(`/api/invoices/${currentInvoiceId}/status`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  });
  viewInvoice(currentInvoiceId);
  loadInvoices();
}

document.getElementById('mark-sent-btn').addEventListener('click', () => updateStatus('sent'));
document.getElementById('mark-paid-btn').addEventListener('click', () => updateStatus('paid'));

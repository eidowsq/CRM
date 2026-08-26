const state = {
  customers: [],
  activities: [],
  contacts: [],
  contracts: [],
  payments: [],
  contractProducts: [],
  users: [],
  customerPage: 1,
  customerPageSize: 10,
  customerFilter: { stage: "", keyword: "", customerStage: "" },
  selectedCustomerId: null,
  editingCustomerId: null,
  editingUserId: null,
  detailBasicEditing: false,
};

const $ = (id) => document.getElementById(id);

const regionOptions = {
  "北京市": { "北京市": ["东城区", "西城区", "朝阳区", "海淀区", "丰台区", "昌平区"] },
  "上海市": { "上海市": ["黄浦区", "徐汇区", "长宁区", "浦东新区", "闵行区", "宝山区"] },
  "广东省": {
    "广州市": ["天河区", "越秀区", "海珠区", "白云区", "番禺区", "黄埔区"],
    "深圳市": ["南山区", "福田区", "罗湖区", "宝安区", "龙岗区", "龙华区"],
    "东莞市": ["南城街道", "东城街道", "万江街道", "长安镇", "虎门镇", "常平镇"],
  },
  "浙江省": {
    "杭州市": ["西湖区", "拱墅区", "上城区", "滨江区", "余杭区"],
    "宁波市": ["海曙区", "江北区", "鄞州区", "镇海区", "北仑区"],
  },
};

if (!localStorage.getItem("crm_token")) {
  location.replace("/login.html");
}

const currentUser = localStorage.getItem("crm_user") || "";
const currentRole = localStorage.getItem("crm_role") || (currentUser === "admin" ? "admin" : "user");
const currentAlias = localStorage.getItem("crm_alias") || currentUser || "用户";

const customerSourceOptions = [
  "邮件咨询", "电话咨询", "个人资源", "展会资源", "公司资源", "招商资源", "陌拜",
  "预约上门", "线上询价", "线上注册", "转介绍", "广告", "搜索引擎", "促销活动",
];

const customerLevelOptions = ["重点", "普通", "低优先级"];
const customerStageOptions = ["未成交", "潜在客户", "洽谈中", "方案报价", "已成交", "暂不跟进"];
const followupTypeOptions = ["打电话", "微信沟通", "上门拜访", "方案沟通"];
const overviewMetricDefs = [
  { key: "month_deal_amount", label: "本月成交金额", type: "money" },
  { key: "year_deal_amount", label: "本年度成交金额", type: "money" },
  { key: "month_deal_count", label: "本月成交单数", type: "count" },
  { key: "year_deal_count", label: "本年成交单数", type: "count" },
  { key: "month_new_contacts", label: "本月新增联系人", type: "count" },
  { key: "year_new_contacts", label: "本年新增联系人", type: "count" },
  { key: "total_deals", label: "总成交", type: "count" },
  { key: "total_contacts", label: "总联系人", type: "count" },
  { key: "total_deal_amount", label: "总成交金额", type: "money" },
  { key: "month_payment_amount", label: "本月回款金额", type: "money" },
  { key: "year_payment_amount", label: "本年回款金额", type: "money" },
  { key: "total_payment_amount", label: "总回款金额", type: "money" },
  { key: "month_pending_amount", label: "本月待回款金额", type: "money" },
  { key: "year_pending_amount", label: "本年待回款金额", type: "money" },
  { key: "total_pending_amount", label: "总待回款金额", type: "money" },
];

function textOrDash(value) {
  const text = String(value ?? "").trim();
  return text || "-";
}

function money(value) {
  return `¥${Number(value || 0).toLocaleString("zh-CN", { minimumFractionDigits: 0, maximumFractionDigits: 2 })}`;
}

function dateText(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleDateString("zh-CN");
}

function dateTimeText(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString("zh-CN");
}

function toDateInputValue(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

function toAsiaShanghaiDateTime(date, time = "00:00:00") {
  return date ? `${date}T${time}+08:00` : "";
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

async function api(path, options = {}) {
  const headers = {
    "Content-Type": "application/json",
    "X-CRM-USER": currentUser,
    "X-CRM-ROLE": currentRole,
    ...(options.headers || {}),
  };
  const response = await fetch(path, { ...options, headers });
  const body = await response.json();
  if (!response.ok) {
    throw new Error(body.error || "请求失败");
  }
  return body.data;
}

async function loadUiConfig() {
  try {
    const config = await api("/api/app-config");
    const brand = String(config?.brand_name || "northstar").trim();
    const workspace = String(config?.workspace_name || "销售工作台").trim();
    if ($("brand-name")) $("brand-name").innerHTML = `${escapeHtml(brand)}<span class="dot">.</span>`;
    if ($("workspace-name")) $("workspace-name").textContent = workspace || "销售工作台";
    document.title = `${brand || "northstar"} CRM`;
  } catch {
    // keep existing defaults when config is unavailable
  }
}

function setImportProgress(percent, text) {
  if ($("import-progress-fill")) $("import-progress-fill").style.width = `${Math.max(0, Math.min(100, percent))}%`;
  if ($("import-progress-percent")) $("import-progress-percent").textContent = `${Math.round(percent)}%`;
  if ($("import-progress-text") && text) $("import-progress-text").textContent = text;
}

function validateAccountPassword(password) {
  const value = String(password || "");
  if (value.length < 8) return "密码至少需要 8 位";
  if (!/[A-Z]/.test(value)) return "密码必须包含大写字母";
  if (!/[a-z]/.test(value)) return "密码必须包含小写字母";
  if (!/[0-9]/.test(value)) return "密码必须包含数字";
  if (!/[^A-Za-z0-9]/.test(value)) return "密码必须包含特殊字符";
  return "";
}

function bindPasswordToggle(toggleId, inputId) {
  const toggle = $(toggleId);
  const input = $(inputId);
  if (!toggle || !input) return;
  toggle.addEventListener("change", () => {
    input.type = toggle.checked ? "text" : "password";
  });
}

function showImportProgressModal() {
  $("import-progress-modal")?.classList.remove("hidden");
  $("import-progress-summary")?.classList.add("hidden");
  $("import-error-panel")?.classList.add("hidden");
  if ($("import-error-list")) $("import-error-list").innerHTML = "";
  setImportProgress(0, "正在准备导入...");
}

function hideImportProgressModal() {
  $("import-progress-modal")?.classList.add("hidden");
}

function renderImportErrors(errors) {
  if (!$("import-error-list")) return;
  if (!Array.isArray(errors) || !errors.length) {
    $("import-error-panel")?.classList.add("hidden");
    $("import-error-list").innerHTML = "";
    return;
  }
  $("import-error-panel")?.classList.remove("hidden");
  $("import-error-list").innerHTML = errors.map((item) => `
    <div class="import-error-item">
      <strong>第 ${escapeHtml(item.row ?? "-")} 行</strong>
      <span>${escapeHtml(item.name || "未命名客户")}</span>
      <small>${escapeHtml(item.reason || "导入失败")}</small>
    </div>
  `).join("");
}

async function importCustomers(file, onProgress) {
  const form = new FormData();
  form.append("file", file);
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/customers/import");
    xhr.setRequestHeader("X-CRM-USER", currentUser);
    xhr.setRequestHeader("X-CRM-ROLE", currentRole);
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable && typeof onProgress === "function") {
        onProgress(Math.min((event.loaded / event.total) * 90, 90), "正在上传文件...");
      }
    };
    xhr.upload.onload = () => {
      if (typeof onProgress === "function") onProgress(95, "文件上传完成，正在处理导入...");
    };
    xhr.onreadystatechange = () => {
      if (xhr.readyState !== 4) return;
      try {
        const body = JSON.parse(xhr.responseText || "{}");
        if (xhr.status >= 200 && xhr.status < 300) {
          if (typeof onProgress === "function") onProgress(100, "导入完成");
          resolve(body.data);
        } else {
          reject(new Error(body.error || "导入失败"));
        }
      } catch (err) {
        reject(new Error("导入失败"));
      }
    };
    xhr.onerror = () => reject(new Error("导入失败"));
    if (typeof onProgress === "function") onProgress(10, "正在上传文件...");
    xhr.send(form);
  });
}

async function exportCustomers() {
  const response = await fetch("/api/customers/export", {
    headers: {
      "X-CRM-USER": currentUser,
      "X-CRM-ROLE": currentRole,
    },
  });
  if (!response.ok) {
    throw new Error("导出失败");
  }
  const blob = await response.blob();
  const disposition = response.headers.get("Content-Disposition") || "";
  const match = disposition.match(/filename="([^"]+)"/);
  const filename = match ? match[1] : `customers_${Date.now()}.csv`;
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

async function createContact(payload) {
  return api("/api/contacts", { method: "POST", body: JSON.stringify(payload) });
}

async function createContract(payload) {
  const response = await fetch("/api/contracts", {
    method: "POST",
    body: payload,
    headers: {
      "X-CRM-USER": currentUser,
      "X-CRM-ROLE": currentRole,
    },
  });
  const body = await response.json();
  if (!response.ok) {
    throw new Error(body.error || "合同创建失败");
  }
  return body.data;
}

async function reviewContract(id, payload) {
  return api(`/api/contracts/${id}/review`, { method: "POST", body: JSON.stringify(payload) });
}

async function createUser(payload) {
  return api("/api/users", { method: "POST", body: JSON.stringify(payload) });
}

async function updateUser(id, payload) {
  return api(`/api/users/${id}`, { method: "PUT", body: JSON.stringify(payload) });
}

async function changeOwnPassword(payload) {
  return api("/api/auth/change-password", { method: "POST", body: JSON.stringify(payload) });
}

async function createPayment(payload) {
  return api("/api/payments", { method: "POST", body: JSON.stringify(payload) });
}

async function reviewPayment(id, payload) {
  return api(`/api/payments/${id}/review`, { method: "POST", body: JSON.stringify(payload) });
}

async function transferCustomerToPool(customerId) {
  return api(`/api/customers/${customerId}/to-pool`, { method: "POST", body: "{}" });
}

async function claimCustomerFromPool(customerId) {
  return api(`/api/customers/${customerId}/claim`, { method: "POST", body: "{}" });
}

function getCustomerById(id) {
  return (state.customers || []).find((item) => Number(item.id) === Number(id));
}

function getCustomerPhone(customer) {
  return textOrDash(customer.phone);
}

function isPoolCustomer(customer) {
  const owner = String(customer.owner || "").trim();
  return owner === "" || owner === "未分配" || owner === "-";
}

function customerNameButton(customer) {
  const id = Number(customer.id || 0);
  return `<button class="customer-link" type="button" data-open-detail="${id}" onclick="window.openCustomerDetailById?.(${id})"><span class="name-tip">${textOrDash(customer.name)}<span class="phone-pop">电话：${escapeHtml(getCustomerPhone(customer))}</span></span></button>`;
}

function overviewRow(customer) {
  return `<tr>
    <td><div class="customer-name"><span class="company-icon">${textOrDash(customer.name).slice(0, 1)}</span>${customerNameButton(customer)}</div></td>
    <td class="muted">${textOrDash(customer.industry)}</td>
    <td>${textOrDash(customer.owner)}</td>
    <td><span class="stage ${escapeHtml(textOrDash(customer.stage))}">${textOrDash(customer.stage)}</span></td>
    <td class="muted">${dateText(customer.next_contact)}</td>
    <td class="muted">${dateTimeText(customer.updated_at)}</td>
  </tr>`;
}

function detailRow(customer) {
  return `<tr>
    <td>${customerNameButton(customer)}</td>
    <td>${textOrDash(customer.level)}</td>
    <td>${textOrDash(customer.industry)}</td>
    <td>${textOrDash(customer.source)}</td>
    <td><span class="stage ${escapeHtml(textOrDash(customer.stage))}">${textOrDash(customer.stage)}</span></td>
    <td>${textOrDash(customer.phone)}</td>
    <td class="wrap-cell">${textOrDash(customer.follow_up_record)}</td>
    <td>${textOrDash(customer.website)}</td>
    <td>${dateText(customer.next_contact)}</td>
    <td class="wrap-cell">${textOrDash(customer.note)}</td>
    <td>${textOrDash(customer.creator)}</td>
    <td>${dateTimeText(customer.updated_at)}</td>
    <td>${dateTimeText(customer.created_at)}</td>
    <td>${textOrDash(customer.owner)}</td>
    <td>${textOrDash(customer.province)}</td>
    <td>${textOrDash(customer.city)}</td>
    <td>${textOrDash(customer.district)}</td>
    <td class="wrap-cell">${textOrDash(customer.address)}</td>
  </tr>`;
}

function poolRow(customer) {
  return detailRow(customer);
}

function todoRow(customer) {
  return `<tr>
    <td><div class="customer-name"><span class="company-icon">${textOrDash(customer.name).slice(0, 1)}</span>${customerNameButton(customer)}</div></td>
    <td>${textOrDash(customer.owner)}</td>
    <td><span class="stage ${escapeHtml(textOrDash(customer.stage))}">${textOrDash(customer.stage)}</span></td>
    <td>${textOrDash(customer.phone)}</td>
    <td>${dateText(customer.next_contact)}</td>
    <td>${textOrDash(customer.creator)}</td>
  </tr>`;
}

function getFilteredCustomers() {
  const lowerKeyword = String(state.customerFilter.keyword || "").toLowerCase();
  return (state.customers || []).filter((customer) => {
    if (isPoolCustomer(customer)) return false;
    if (state.customerFilter.stage && customer.stage !== state.customerFilter.stage) return false;
    if (state.customerFilter.customerStage && customer.stage !== state.customerFilter.customerStage) return false;
    if (!lowerKeyword) return true;
    return [customer.name, customer.industry, customer.owner, customer.source, customer.phone, customer.address]
      .some((item) => String(item || "").toLowerCase().includes(lowerKeyword));
  });
}

function getPoolCustomers(keyword = "") {
  const lowerKeyword = keyword.toLowerCase();
  return (state.customers || []).filter((customer) => {
    if (!isPoolCustomer(customer)) return false;
    if (!lowerKeyword) return true;
    return [customer.name, customer.industry, customer.source, customer.phone].some((item) => String(item || "").toLowerCase().includes(lowerKeyword));
  });
}

function isTodoCustomer(customer) {
  if (!customer.next_contact) return false;
  const now = new Date();
  const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const next = new Date(customer.next_contact);
  return next >= start && next <= new Date(start.getFullYear(), start.getMonth(), start.getDate(), 23, 59, 59);
}

function getTodoCustomers(keyword = "") {
  const lowerKeyword = keyword.toLowerCase();
  return (state.customers || []).filter((customer) => {
    if (isPoolCustomer(customer)) return false;
    if (!isTodoCustomer(customer)) return false;
    if (!lowerKeyword) return true;
    return [customer.name, customer.owner, customer.phone, customer.creator].some((item) => String(item || "").toLowerCase().includes(lowerKeyword));
  });
}

function getCustomerActivities(customerId) {
  return (state.activities || [])
    .filter((item) => item && item.customer && Number(item.customer.id) === Number(customerId))
    .sort((a, b) => new Date(b.created_at || 0) - new Date(a.created_at || 0));
}

function getCustomerContacts(customerId) {
  return (state.contacts || [])
    .filter((item) => item && item.customer && Number(item.customer.id) === Number(customerId));
}

function getCustomerContracts(customerId) {
  return (state.contracts || [])
    .filter((item) => item && item.customer && Number(item.customer.id) === Number(customerId))
    .sort((a, b) => new Date(b.created_at || 0) - new Date(a.created_at || 0));
}

function getCustomerPayments(customerId) {
  return (state.payments || [])
    .filter((item) => item && item.customer && Number(item.customer.id) === Number(customerId))
    .sort((a, b) => new Date(b.payment_date || 0) - new Date(a.payment_date || 0));
}

function fillPaymentCustomerOptions(selectedId = 0) {
  const list = $("payment-customer-options");
  const hiddenInput = $("payment-customer-id");
  const trigger = $("payment-customer-trigger");
  if (!list || !hiddenInput || !trigger) return;
  const customers = (state.customers || []).filter((item) => !isPoolCustomer(item));
  list.innerHTML = customers.length
    ? customers.map((item) => `<button class="payment-customer-option" type="button" data-payment-customer-option="${item.id}" data-payment-customer-name="${escapeHtml(textOrDash(item.name))}">${escapeHtml(textOrDash(item.name))}</button>`).join("")
    : `<div class="payment-customer-empty">暂无可选客户</div>`;
  hiddenInput.value = selectedId ? String(selectedId) : "";
  const selectedCustomer = customers.find((item) => Number(item.id) === Number(selectedId));
  trigger.textContent = selectedCustomer ? textOrDash(selectedCustomer.name) : "请选择客户名称";
}

function filterPaymentCustomerOptions(keyword = "") {
  const lowerKeyword = String(keyword || "").trim().toLowerCase();
  document.querySelectorAll("[data-payment-customer-option]").forEach((option) => {
    const name = String(option.getAttribute("data-payment-customer-name") || "").toLowerCase();
    option.classList.toggle("hidden", !!lowerKeyword && !name.includes(lowerKeyword));
  });
}

function fillContractCustomerOptions(selectedId = 0) {
  const list = $("contract-customer-options");
  const hiddenInput = $("contract-customer");
  const trigger = $("contract-customer-trigger");
  if (!list || !hiddenInput || !trigger) return;
  const customers = (state.customers || []).filter((item) => !isPoolCustomer(item));
  list.innerHTML = customers.length
    ? customers.map((item) => `<button class="payment-customer-option" type="button" data-contract-customer-option="${item.id}" data-contract-customer-name="${escapeHtml(textOrDash(item.name))}">${escapeHtml(textOrDash(item.name))}</button>`).join("")
    : `<div class="payment-customer-empty">暂无可选客户</div>`;
  hiddenInput.value = selectedId ? String(selectedId) : "";
  const selectedCustomer = customers.find((item) => Number(item.id) === Number(selectedId));
  trigger.textContent = selectedCustomer ? textOrDash(selectedCustomer.name) : "请选择客户名称";
}

function filterContractCustomerOptions(keyword = "") {
  const lowerKeyword = String(keyword || "").trim().toLowerCase();
  document.querySelectorAll("[data-contract-customer-option]").forEach((option) => {
    const name = String(option.getAttribute("data-contract-customer-name") || "").toLowerCase();
    option.classList.toggle("hidden", !!lowerKeyword && !name.includes(lowerKeyword));
  });
}

function getContractDisplayNumber(contract) {
  if (!contract) return "";
  return textOrDash(contract.serial_no || contract.title || contract.id);
}

function contractStatusText(status) {
  if (status === "approved") return "已通过";
  if (status === "rejected") return "已驳回";
  return "待审批";
}

function paymentStatusText(status) {
  if (status === "approved") return "已通过";
  if (status === "rejected") return "已驳回";
  return "待审核";
}

function renderOverviewMetrics(summary = {}) {
  const container = $("overview-metrics");
  if (!container) return;
  const sparklineClasses = ["green", "blue-line", "peach-line", "violet-line"];
  container.innerHTML = overviewMetricDefs.map((metric, index) => {
    const value = summary?.[metric.key] ?? 0;
    const displayValue = metric.type === "money" ? money(value) : value;
    return `
      <article class="metric">
        <div class="metric-label">${metric.label}</div>
        <strong>${displayValue}</strong>
        <small>${currentRole === "admin" ? "显示全部数据" : "显示当前账号数据"}</small>
        <div class="sparkline ${sparklineClasses[index % sparklineClasses.length]}"></div>
      </article>
    `;
  }).join("");
}

function updateContractBadge() {
  const badge = $("contract-total");
  if (!badge) return;
  const pendingCount = (state.contracts || []).filter((item) => item && item.status === "pending").length;
  badge.textContent = pendingCount;
}

function updatePaymentApprovalBadge() {
  const badge = $("payment-approval-total");
  if (!badge) return;
  const pendingCount = (state.payments || []).filter((item) => item && item.status === "pending").length;
  badge.textContent = pendingCount;
}

function parseContractAttachments(value) {
  if (!value) return [];
  if (Array.isArray(value)) return value;
  try {
    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function parseContractProducts(value) {
  if (!value) return [];
  if (Array.isArray(value)) return value;
  try {
    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function createEmptyContractProduct() {
  return {
    name: "",
    category: "",
    unit: "",
    standard_price: 0,
    sale_price: 0,
    quantity: 1,
    discount: 100,
  };
}

function normalizeContractProduct(product = {}) {
  const standardPrice = Number(product.standard_price || 0);
  const salePrice = Number(product.sale_price || 0);
  const quantity = Number(product.quantity || 0);
  const discount = Number(product.discount || 0);
  return {
    name: String(product.name || "").trim(),
    category: String(product.category || "").trim(),
    unit: String(product.unit || "").trim(),
    standard_price: Number.isFinite(standardPrice) ? standardPrice : 0,
    sale_price: Number.isFinite(salePrice) ? salePrice : 0,
    quantity: Number.isFinite(quantity) ? quantity : 0,
    discount: Number.isFinite(discount) ? discount : 0,
  };
}

function contractProductTotal(product) {
  const normalized = normalizeContractProduct(product);
  return normalized.sale_price * normalized.quantity * (normalized.discount || 0) / 100;
}

function renderContractProducts() {
  const body = $("contract-product-body");
  const empty = $("contract-product-empty");
  if (!body || !empty) return;
  body.innerHTML = (state.contractProducts || []).map((product, index) => `
    <div class="contract-product-row contract-product-body-row">
      <input data-contract-product="${index}" data-field="name" value="${escapeHtml(product.name)}" placeholder="输入产品名称">
      <input data-contract-product="${index}" data-field="category" value="${escapeHtml(product.category)}" placeholder="输入类别">
      <input data-contract-product="${index}" data-field="unit" value="${escapeHtml(product.unit)}" placeholder="台/件">
      <input data-contract-product="${index}" data-field="standard_price" type="number" step="0.01" min="0" value="${product.standard_price || 0}">
      <input data-contract-product="${index}" data-field="sale_price" type="number" step="0.01" min="0" value="${product.sale_price || 0}">
      <input data-contract-product="${index}" data-field="quantity" type="number" step="1" min="0" value="${product.quantity || 0}">
      <input data-contract-product="${index}" data-field="discount" type="number" step="0.01" min="0" value="${product.discount || 0}">
      <div class="readonly-cell contract-product-total" data-contract-product-total="${index}">${money(contractProductTotal(product))}</div>
      <button class="contract-product-remove" type="button" data-remove-contract-product="${index}">删除</button>
    </div>
  `).join("");
  empty.classList.toggle("hidden", state.contractProducts.length > 0);
  syncContractAmountFromProducts();
}

function addContractProductRow() {
  state.contractProducts = [...(state.contractProducts || []), createEmptyContractProduct()];
  renderContractProducts();
}

function syncContractAmountFromProducts() {
  const amountInput = $("contract-form")?.elements?.amount;
  if (!amountInput) return;
  const total = (state.contractProducts || []).reduce((sum, item) => sum + contractProductTotal(item), 0);
  if (total > 0) {
    amountInput.value = total.toFixed(2);
  }
}

function renderCustomerTable(list) {
  const totalPages = Math.max(1, Math.ceil(list.length / state.customerPageSize));
  state.customerPage = Math.min(Math.max(state.customerPage, 1), totalPages);
  const start = (state.customerPage - 1) * state.customerPageSize;
  const end = Math.min(start + state.customerPageSize, list.length);
  if ($("customer-list-2")) $("customer-list-2").innerHTML = list.slice(start, end).map(detailRow).join("");
  if ($("customer-page-summary")) $("customer-page-summary").textContent = list.length ? `显示第 ${start + 1}-${end} 条，共 ${list.length} 条` : "暂无客户数据";
  if ($("customer-page-current")) $("customer-page-current").textContent = `${state.customerPage} / ${totalPages}`;
  if ($("customer-page-input")) $("customer-page-input").max = totalPages;
  if ($("customer-prev")) $("customer-prev").disabled = state.customerPage <= 1;
  if ($("customer-next")) $("customer-next").disabled = state.customerPage >= totalPages;
}

function renderPoolCustomers() {
  const rows = getPoolCustomers($("pool-search")?.value || "");
  if ($("pool-list")) $("pool-list").innerHTML = rows.map(poolRow).join("");
  if ($("pool-empty")) $("pool-empty").classList.toggle("hidden", rows.length > 0);
  if ($("pool-total")) $("pool-total").textContent = rows.length;
}

function renderTodoCustomers() {
  const rows = getTodoCustomers($("todo-search")?.value || "");
  if ($("todo-list")) $("todo-list").innerHTML = rows.map(todoRow).join("");
  if ($("todo-empty")) $("todo-empty").classList.toggle("hidden", rows.length > 0);
  if ($("todo-total")) $("todo-total").textContent = rows.length;
  if ($("todo-summary")) $("todo-summary").textContent = `当天需要联系的客户：${rows.length} 个`;
}

function renderCustomers() {
  const filtered = getFilteredCustomers();
  if ($("customer-list")) $("customer-list").innerHTML = filtered.slice(0, 6).map(overviewRow).join("");
  if ($("empty")) $("empty").classList.toggle("hidden", filtered.length > 0);
  if ($("all-count")) $("all-count").textContent = state.customers.length;
  if ($("nav-total")) $("nav-total").textContent = state.customers.length;
  renderCustomerTable(filtered);
  renderPoolCustomers();
  renderTodoCustomers();
  fillContractCustomerOptions();
}

function fillCustomerStageFilter() {
  if (!$("customer-stage-filter")) return;
  $("customer-stage-filter").innerHTML = ['<option value="">全部状态</option>']
    .concat(customerStageOptions.map((item) => `<option value="${escapeHtml(item)}">${escapeHtml(item)}</option>`))
    .join("");
  $("customer-stage-filter").value = state.customerFilter.customerStage || "";
}

function renderActivities() {
  const list = state.activities || [];
  if (!$("activity-list")) return;
  $("activity-list").innerHTML = list.length
    ? list.map((item) => `<div class="activity-item"><span class="activity-dot">◷</span><div><strong>${textOrDash(item.customer ? item.customer.name : "客户")} · ${textOrDash(item.type || "跟进")}</strong><p>${textOrDash(item.content)}</p><time>${dateTimeText(item.created_at)}</time></div></div>`).join("")
    : '<div class="empty">还没有跟进记录。</div>';
}

function renderContracts() {
  const list = state.contracts || [];
  if (!$("contract-list")) return;
  $("contract-list").innerHTML = list.map((item) => `
    <tr>
      <td><button class="customer-link" type="button" data-open-contract="${item.id}">${textOrDash(item.title)}</button></td>
      <td>${textOrDash(item.customer ? item.customer.name : "")}</td>
      <td>${money(item.amount)}</td>
      <td><span class="contract-status contract-status-${item.status || "pending"}">${contractStatusText(item.status)}</span></td>
      <td>${textOrDash(item.submitter)}</td>
      <td>${textOrDash(item.reviewer)}</td>
      <td>${dateTimeText(item.created_at)}</td>
      <td>
        ${currentRole === "admin" && item.status === "pending"
          ? `<button class="secondary mini-btn" type="button" data-contract-review="${item.id}" data-action="approve">通过</button> <button class="secondary mini-btn" type="button" data-contract-review="${item.id}" data-action="reject">驳回</button>`
          : `<span class="muted">${contractStatusText(item.status)}</span>`}
      </td>
    </tr>
  `).join("");
  if ($("contract-empty")) $("contract-empty").classList.toggle("hidden", list.length > 0);
  updateContractBadge();
}

function renderPaymentApprovals() {
  const list = state.payments || [];
  if (!$("payment-approval-list") || !$("payment-approval-empty")) return;
  $("payment-approval-list").innerHTML = list.map((item) => `
    <tr>
      <td>${textOrDash(item.serial_no)}</td>
      <td>${textOrDash(item.customer ? item.customer.name : "")}</td>
      <td><span class="contract-status contract-status-${item.status || "pending"}">${paymentStatusText(item.status)}</span></td>
      <td><button class="customer-link" type="button" data-open-contract="${item.contract ? item.contract.id : 0}">${textOrDash(item.contract_title || (item.contract ? item.contract.title : ""))}</button></td>
      <td>${money(item.contract_amount || (item.contract ? item.contract.amount : 0))}</td>
      <td>${money(item.payment_amount)}</td>
      <td>${textOrDash(item.payment_method)}</td>
      <td>${dateText(item.payment_date)}</td>
      <td>
        ${currentRole === "admin" && item.status === "pending"
          ? `<button class="secondary mini-btn" type="button" data-payment-review="${item.id}" data-action="approve">通过</button> <button class="secondary mini-btn" type="button" data-payment-review="${item.id}" data-action="reject">驳回</button>`
          : `<span class="muted">${paymentStatusText(item.status)}</span>`}
      </td>
    </tr>
  `).join("");
  $("payment-approval-empty").classList.toggle("hidden", list.length > 0);
  updatePaymentApprovalBadge();
}

function getContractById(id) {
  return (state.contracts || []).find((item) => Number(item.id) === Number(id));
}

function openContractDetailModal(contractId) {
  const contract = getContractById(contractId);
  if (!contract) return;
  const contractBackdrop = $("contract-detail-modal");
  const contractModal = contractBackdrop?.querySelector(".contract-detail-modal");
  $("contract-detail-avatar").textContent = textOrDash(contract.title).slice(0, 1);
  $("contract-detail-title").textContent = textOrDash(contract.title);
  $("contract-detail-customer").textContent = `客户：${textOrDash(contract.customer ? contract.customer.name : "")}`;
  const summary = document.querySelector(".contract-detail-summary");
  if (summary) {
    summary.innerHTML = [
      ["合同编号", contract.serial_no],
      ["商机名称", contract.business_name],
      ["客户名称", contract.customer ? contract.customer.name : ""],
      ["合同金额", money(contract.amount)],
      ["下单时间", dateText(contract.order_date)],
      ["合同开始时间", dateText(contract.start_date)],
      ["到期时间", dateText(contract.end_date)],
      ["客户签约人", contract.customer_signer],
      ["公司签约人", contract.company_signer],
    ].map(([label, value]) => `<div><span>${label}</span><strong>${textOrDash(value)}</strong></div>`).join("");
  }

  const gridEntries = [
    ["合同编号", contract.serial_no],
    ["合同名称", contract.title],
    ["客户名称", contract.customer ? contract.customer.name : ""],
    ["商机名称", contract.business_name],
    ["合同金额", money(contract.amount)],
    ["下单时间", dateText(contract.order_date)],
    ["合同开始时间", dateText(contract.start_date)],
    ["合同到期时间", dateText(contract.end_date)],
    ["客户签约人", contract.customer_signer],
    ["公司签约人", contract.company_signer],
    ["合同状态", contractStatusText(contract.status)],
    ["提交人", contract.submitter],
    ["审批人", contract.reviewer],
    ["提交时间", dateTimeText(contract.created_at)],
    ["审批时间", dateTimeText(contract.reviewed_at)],
  ];
  $("contract-detail-grid").innerHTML = gridEntries
    .map(([label, value]) => `<div class="detail-card"><span>${label}</span><strong>${textOrDash(value)}</strong></div>`)
    .join("");

  const products = parseContractProducts(contract.products);
  $("contract-detail-products").innerHTML = products.map((product) => {
    const item = normalizeContractProduct(product);
    return `
      <div class="contract-product-row contract-detail-product-row">
        <div class="readonly-cell">${textOrDash(item.name)}</div>
        <div class="readonly-cell">${textOrDash(item.category)}</div>
        <div class="readonly-cell">${textOrDash(item.unit)}</div>
        <div class="readonly-cell">${money(item.standard_price)}</div>
        <div class="readonly-cell">${money(item.sale_price)}</div>
        <div class="readonly-cell">${item.quantity}</div>
        <div class="readonly-cell">${item.discount}</div>
        <div class="readonly-cell contract-product-total">${money(contractProductTotal(item))}</div>
        <div class="readonly-cell">-</div>
      </div>
    `;
  }).join("");
  $("contract-detail-products-empty").classList.toggle("hidden", products.length > 0);

  const attachments = parseContractAttachments(contract.attachments);
  $("contract-detail-attachments").innerHTML = attachments
    .map((attachment) => `<a class="contract-attachment-link" href="${attachment.url}" target="_blank" rel="noreferrer">${attachment.name}</a>`)
    .join("");
  $("contract-detail-attachments-empty").classList.toggle("hidden", attachments.length > 0);

  $("contract-detail-note").textContent = textOrDash(contract.content);
  contractBackdrop?.classList.remove("hidden");
  if (contractBackdrop) contractBackdrop.scrollTop = 0;
  if (contractModal) contractModal.scrollTop = 0;
}

function closeContractDetailModal() {
  $("contract-detail-modal").classList.add("hidden");
}

function renderUsers() {
  const list = state.users || [];
  if (!$("accounts-list") || !$("accounts-empty")) return;
  $("accounts-list").innerHTML = list.map((user) => `
    <tr>
      <td>${textOrDash(user.username)}</td>
      <td>${textOrDash(user.alias)}</td>
      <td>${textOrDash(user.role)}</td>
      <td>${dateTimeText(user.created_at)}</td>
      <td>
        <button class="secondary mini-btn" type="button" data-edit-user="${user.id}">编辑</button>
        ${user.username === "admin" ? "" : `<button class="secondary mini-btn" type="button" data-delete-user="${user.id}">删除</button>`}
      </td>
    </tr>
  `).join("");
  $("accounts-empty").classList.toggle("hidden", list.length > 0);
}

function renderDetailTabs(tabName = "followups") {
  document.querySelectorAll(".detail-tab").forEach((tab) => tab.classList.toggle("active", tab.dataset.tab === tabName));
  const detailModal = $("detail-modal");
  if (!detailModal) return;
  detailModal.querySelectorAll(".detail-pane").forEach((pane) => pane.classList.add("hidden"));
  detailModal.querySelector(`#pane-${tabName}`)?.classList.remove("hidden");
}

function renderDetailBasic(customer) {
  const editButton = $("detail-basic-edit");
  const saveButton = $("detail-basic-save");
  const cancelButton = $("detail-basic-cancel");
  if (!customer) return;
  if (state.detailBasicEditing) {
    editButton?.classList.add("hidden");
    saveButton?.classList.remove("hidden");
    cancelButton?.classList.remove("hidden");
    $("detail-basic-grid").innerHTML = `
      <label class="detail-edit-field"><span>客户名称</span><input id="detail-edit-name" value="${escapeHtml(customer.name)}"></label>
      <label class="detail-edit-field"><span>客户级别</span><select id="detail-edit-level">${customerLevelOptions.map((item) => `<option ${customer.level === item ? "selected" : ""}>${item}</option>`).join("")}</select></label>
      <label class="detail-edit-field"><span>客户行业</span><input id="detail-edit-industry" value="${escapeHtml(customer.industry)}"></label>
      <label class="detail-edit-field"><span>客户来源</span><select id="detail-edit-source"><option value="">请选择客户来源</option>${customerSourceOptions.map((item) => `<option ${customer.source === item ? "selected" : ""}>${item}</option>`).join("")}</select></label>
      <label class="detail-edit-field"><span>成交状态</span><select id="detail-edit-stage">${customerStageOptions.map((item) => `<option ${customer.stage === item ? "selected" : ""}>${item}</option>`).join("")}</select></label>
      <label class="detail-edit-field"><span>电话</span><input id="detail-edit-phone" value="${escapeHtml(customer.phone)}"></label>
      <label class="detail-edit-field"><span>网址</span><input id="detail-edit-website" value="${escapeHtml(customer.website)}"></label>
      <label class="detail-edit-field"><span>负责人</span><input id="detail-edit-owner" value="${escapeHtml(customer.owner)}"></label>
      <label class="detail-edit-field"><span>省</span><input id="detail-edit-province" value="${escapeHtml(customer.province)}"></label>
      <label class="detail-edit-field"><span>市</span><input id="detail-edit-city" value="${escapeHtml(customer.city)}"></label>
      <label class="detail-edit-field"><span>区/县</span><input id="detail-edit-district" value="${escapeHtml(customer.district)}"></label>
      <label class="detail-edit-field detail-edit-span2"><span>详细地址</span><input id="detail-edit-address" value="${escapeHtml(customer.address)}"></label>
      <label class="detail-edit-field"><span>下次联系时间</span><input id="detail-edit-next-contact" type="date" value="${toDateInputValue(customer.next_contact)}"></label>
      <label class="detail-edit-field detail-edit-span2"><span>备注</span><textarea id="detail-edit-note" rows="4">${escapeHtml(customer.note)}</textarea></label>
    `;
    return;
  }
  editButton?.classList.remove("hidden");
  saveButton?.classList.add("hidden");
  cancelButton?.classList.add("hidden");
  const entries = [
    ["客户名称", customer.name],
    ["客户级别", customer.level],
    ["客户行业", customer.industry],
    ["客户来源", customer.source],
    ["成交状态", customer.stage],
    ["电话", customer.phone],
    ["网址", customer.website],
    ["负责人", customer.owner],
    ["创建人", customer.creator],
    ["省", customer.province],
    ["市", customer.city],
    ["区/县", customer.district],
    ["详细地址", customer.address],
    ["下次联系时间", dateText(customer.next_contact)],
    ["创建时间", dateTimeText(customer.created_at)],
    ["更新时间", dateTimeText(customer.updated_at)],
    ["备注", customer.note],
  ];
  $("detail-basic-grid").innerHTML = entries.map(([label, value]) => `<div class="detail-card"><span>${label}</span><strong>${textOrDash(value)}</strong></div>`).join("");
}

function renderDetailFollowups(customer) {
  const activities = getCustomerActivities(customer.id);
  const items = activities.length ? activities : (customer.follow_up_record ? [{
    type: "跟进记录",
    content: customer.follow_up_record,
    created_at: customer.updated_at || customer.created_at,
    next_action: customer.next_contact,
  }] : []);
  $("detail-followup-list").innerHTML = items.length
    ? items.map((item) => `
      <article class="detail-log-card">
        <div class="detail-log-head">
          <span class="detail-log-avatar">${textOrDash(customer.owner || customer.creator || "客").slice(0, 1)}</span>
          <div><strong>${textOrDash(customer.owner || customer.creator)}</strong><time>${dateTimeText(item.created_at)}</time></div>
          <span class="detail-log-badge">跟进记录</span>
        </div>
        <p>${textOrDash(item.content)}</p>
        <div class="detail-log-tags">
          <span>${textOrDash(item.type)}</span>
          <span>${item.next_action ? dateText(item.next_action) : "-"}</span>
        </div>
      </article>
    `).join("")
    : '<div class="detail-empty">还没有跟进记录，先在上方发布一条吧。</div>';
}

function renderDetailContacts(customer) {
  if (!$("detail-contact-list")) return;
  const contacts = getCustomerContacts(customer.id);
  $("detail-contact-list").innerHTML = contacts.length
    ? contacts.map((item) => `
      <article class="contact-card">
        <strong>${textOrDash(item.name)}</strong>
        <span>${textOrDash(item.role)}</span>
        <span>${textOrDash(item.phone)}</span>
        <span>${textOrDash(item.email)}</span>
      </article>
    `).join("")
    : '<div class="detail-empty">还没有联系人，先新建一个吧。</div>';
}

function renderDetailContracts(customer) {
  const contracts = getCustomerContracts(customer.id);
  $("pane-contract").innerHTML = `
    <div class="section-head page-section payment-head">
      <div><h3>合同信息</h3></div>
      <div class="section-actions"><button class="primary" id="detail-contract-create" type="button">＋ 新建合同</button></div>
    </div>
    ${contracts.length
    ? `<div class="followup-list">${contracts.map((item) => `
        <article class="detail-log-card">
        <div class="detail-log-head">
          <div><button class="customer-link" type="button" data-open-contract="${item.id}">${textOrDash(item.title)}</button><time>${dateTimeText(item.created_at)}</time></div>
          <span class="detail-log-badge">${contractStatusText(item.status)}</span>
          </div>
          <p>金额：${money(item.amount)}</p>
          <p>${textOrDash(item.content)}</p>
          ${parseContractProducts(item.products).length ? `<div class="detail-log-tags">${parseContractProducts(item.products).map((product) => `<span>${textOrDash(product.name)} x ${Number(product.quantity || 0)} / ${money(contractProductTotal(product))}</span>`).join("")}</div>` : ""}
          ${parseContractAttachments(item.attachments).length ? `<div class="detail-log-tags">${parseContractAttachments(item.attachments).map((attachment) => `<a class="contract-attachment-link" href="${attachment.url}" target="_blank" rel="noreferrer">${attachment.name}</a>`).join("")}</div>` : ""}
          <div class="detail-log-tags">
            <span>提交人：${textOrDash(item.submitter)}</span>
            <span>审批人：${textOrDash(item.reviewer)}</span>
          </div>
        </article>
    `).join("")}</div>`
      : '<div class="detail-empty">暂未维护合同信息</div>'}`;
}

function renderDetailPayments(customer) {
  const payments = getCustomerPayments(customer.id);
  const pane = $("pane-payment");
  if (!pane) return;
  pane.innerHTML = `
    <div class="section-head page-section payment-head">
      <div><h3>回款信息</h3></div>
      <div class="section-actions"><button class="primary" id="detail-payment-create" type="button">＋ 新建回款</button></div>
    </div>
    <div class="table-wrap">
      <table class="customer-detail-table payment-table">
        <thead>
          <tr>
            <th>合同名称</th>
            <th>合同金额</th>
            <th>回款金额</th>
            <th>审核状态</th>
            <th>回款日期</th>
          </tr>
        </thead>
        <tbody>
          ${payments.length ? payments.map((item) => `
            <tr>
              <td><button class="customer-link" type="button" data-open-contract="${item.contract ? item.contract.id : 0}">${textOrDash(item.contract_title || (item.contract ? item.contract.title : ""))}</button></td>
              <td>${money(item.contract_amount || (item.contract ? item.contract.amount : 0))}</td>
              <td>${money(item.payment_amount)}</td>
              <td><span class="contract-status contract-status-${item.status || "pending"}">${paymentStatusText(item.status)}</span></td>
              <td>${dateText(item.payment_date)}</td>
            </tr>
          `).join("") : `
            <tr>
              <td colspan="5" class="muted">暂无回款记录</td>
            </tr>
          `}
        </tbody>
      </table>
    </div>
  `;
}

function fillPaymentContractOptions(customerId = state.selectedCustomerId) {
  const select = $("payment-contract-id");
  const customer = getCustomerById(customerId);
  if (!select) return;
  if (!customer) {
    select.innerHTML = `<option value="">请选择合同编号</option>`;
    return;
  }
  const contracts = getCustomerContracts(customer.id);
  select.innerHTML = [`<option value="">请选择合同编号</option>`, ...contracts.map((item) => `<option value="${item.id}">${escapeHtml(getContractDisplayNumber(item))}</option>`)].join("");
}

function openPaymentModal(customerId = state.selectedCustomerId) {
  const customer = getCustomerById(customerId);
  $("payment-form")?.reset();
  fillPaymentCustomerOptions(customer ? customer.id : 0);
  if ($("payment-customer-search")) {
    $("payment-customer-search").value = "";
  }
  filterPaymentCustomerOptions("");
  $("payment-customer-dropdown")?.classList.add("hidden");
  if ($("payment-date")) {
    $("payment-date").value = toDateInputValue(new Date());
  }
  fillPaymentContractOptions(customer ? customer.id : 0);
  $("payment-modal")?.classList.remove("hidden");
}

function closePaymentModal() {
  $("payment-modal")?.classList.add("hidden");
}

function openContractDetailStandalone(contractId) {
  closePaymentModal();
  closeContractModal();
  if (!$("detail-modal")?.classList.contains("hidden")) {
    closeDetailModal();
  }
  openContractDetailModal(contractId);
}

function openDetailModal(customerId) {
  const customer = getCustomerById(customerId);
  if (!customer) return;
  state.selectedCustomerId = customer.id;
  state.detailBasicEditing = false;
  if ($("detail-transfer-pool")) {
    $("detail-transfer-pool").textContent = isPoolCustomer(customer) ? "领用" : "转移到公海池";
  }
  $("detail-avatar").textContent = textOrDash(customer.name).slice(0, 1);
  $("detail-name").textContent = textOrDash(customer.name);
  $("detail-phone-line").textContent = `电话：${getCustomerPhone(customer)}`;
  $("detail-level").textContent = textOrDash(customer.level);
  $("detail-stage").textContent = textOrDash(customer.stage);
  $("detail-owner").textContent = textOrDash(customer.owner);
  $("detail-updated").textContent = dateTimeText(customer.updated_at);
  $("detail-followup-content").value = "";
  $("detail-followup-next").value = "";
  $("detail-followup-type").value = followupTypeOptions[0];
  renderDetailBasic(customer);
  renderDetailFollowups(customer);
  renderDetailContacts(customer);
  renderDetailContracts(customer);
  renderDetailPayments(customer);
  renderDetailTabs("followups");
  $("detail-modal").classList.remove("hidden");
}

window.openCustomerDetailById = function openCustomerDetailById(customerId) {
  const id = Number(customerId);
  if (Number.isFinite(id) && id > 0) openDetailModal(id);
};

function closeDetailModal() {
  $("detail-modal").classList.add("hidden");
  state.selectedCustomerId = null;
  state.detailBasicEditing = false;
}

function openDetailBasicEdit() {
  const customer = getCustomerById(state.selectedCustomerId);
  if (!customer) return;
  state.detailBasicEditing = true;
  renderDetailTabs("basic");
  renderDetailBasic(customer);
}

function cancelDetailBasicEdit() {
  const customer = getCustomerById(state.selectedCustomerId);
  state.detailBasicEditing = false;
  if (customer) renderDetailBasic(customer);
}

async function saveDetailBasicEdit() {
  const customer = getCustomerById(state.selectedCustomerId);
  if (!customer) return;
  const payload = {
    name: $("detail-edit-name").value.trim(),
    level: $("detail-edit-level").value,
    industry: $("detail-edit-industry").value.trim(),
    source: $("detail-edit-source").value,
    stage: $("detail-edit-stage").value,
    phone: $("detail-edit-phone").value.trim(),
    website: $("detail-edit-website").value.trim(),
    owner: $("detail-edit-owner").value.trim(),
    province: $("detail-edit-province").value.trim(),
    city: $("detail-edit-city").value.trim(),
    district: $("detail-edit-district").value.trim(),
    address: $("detail-edit-address").value.trim(),
    note: $("detail-edit-note").value.trim(),
    creator: customer.creator,
    follow_up_record: customer.follow_up_record,
  };
  const nextContactValue = $("detail-edit-next-contact").value;
  if (nextContactValue) {
    payload.next_contact = toAsiaShanghaiDateTime(nextContactValue);
  }
  if (!payload.name) {
    alert("客户名称不能为空");
    return;
  }
  try {
    const updated = await api(`/api/customers/${customer.id}`, { method: "PUT", body: JSON.stringify(payload) });
    const index = state.customers.findIndex((item) => Number(item.id) === Number(updated.id));
    if (index >= 0) state.customers[index] = { ...state.customers[index], ...updated };
    state.detailBasicEditing = false;
    openDetailModal(updated.id);
    renderCustomers();
  } catch (err) {
    alert(err.message);
  }
}

async function submitDetailFollowup() {
  if (!state.selectedCustomerId) return;
  const content = $("detail-followup-content").value.trim();
  if (!content) {
    alert("请输入跟进内容");
    return;
  }
  const payload = {
    customer_id: state.selectedCustomerId,
    type: $("detail-followup-type").value,
    content,
    next_action: $("detail-followup-next").value || "",
  };
  try {
    const created = await api("/api/activities", { method: "POST", body: JSON.stringify(payload) });
    const customer = getCustomerById(state.selectedCustomerId);
    if (customer) {
      created.customer = { id: customer.id, name: customer.name };
      state.activities = [created, ...state.activities];
      customer.follow_up_record = payload.content;
      if (payload.next_action) customer.next_contact = toAsiaShanghaiDateTime(payload.next_action);
      customer.updated_at = new Date().toISOString();
      renderDetailBasic(customer);
      renderDetailFollowups(customer);
    }
    $("detail-followup-content").value = "";
    $("detail-followup-next").value = "";
    renderActivities();
    renderCustomers();
  } catch (err) {
    alert(err.message);
  }
}

async function submitDetailContact() {
  if (!state.selectedCustomerId) return;
  const payload = {
    customer_id: state.selectedCustomerId,
    name: $("detail-contact-name").value.trim(),
    role: $("detail-contact-role").value.trim(),
    phone: $("detail-contact-phone").value.trim(),
    email: $("detail-contact-email").value.trim(),
  };
  if (!payload.name) {
    alert("请输入联系人姓名");
    return;
  }
  try {
    await createContact(payload);
    ["detail-contact-name", "detail-contact-role", "detail-contact-phone", "detail-contact-email"].forEach((id) => { $(id).value = ""; });
    state.contacts = await api("/api/contacts");
    const customer = getCustomerById(state.selectedCustomerId);
    if (customer) renderDetailContacts(customer);
  } catch (err) {
    alert(err.message);
  }
}

async function handleTransferToPool() {
  if (!state.selectedCustomerId) return;
  try {
    const customer = getCustomerById(state.selectedCustomerId);
    const updated = customer && isPoolCustomer(customer)
      ? await claimCustomerFromPool(state.selectedCustomerId)
      : await transferCustomerToPool(state.selectedCustomerId);
    const index = state.customers.findIndex((item) => Number(item.id) === Number(updated.id));
    if (index >= 0) state.customers[index] = { ...state.customers[index], ...updated, owner: "" };
    if (index >= 0 && customer && isPoolCustomer(customer)) {
      state.customers[index] = { ...state.customers[index], ...updated, owner: currentUser };
    }
    closeDetailModal();
    renderCustomers();
  } catch (err) {
    alert(err.message);
  }
}

function fillSelect(select, options, placeholder) {
  select.innerHTML = [`<option value="">${placeholder}</option>`, ...options.map((item) => `<option value="${item}">${item}</option>`)].join("");
}

function enableHorizontalDragScroll(selector) {
  document.querySelectorAll(selector).forEach((container) => {
    if (container.dataset.dragScrollBound === "1") return;
    container.dataset.dragScrollBound = "1";
    let isDown = false;
    let isDragging = false;
    let startX = 0;
    let startScrollLeft = 0;

    container.style.cursor = "grab";
    container.addEventListener("pointerdown", (event) => {
      if (event.button !== 0) return;
      if (event.target.closest?.("button, a, input, select, textarea, label, [role='button']")) return;
      isDown = true;
      isDragging = false;
      startX = event.clientX;
      startScrollLeft = container.scrollLeft;
      container.style.cursor = "grabbing";
    });

    container.addEventListener("pointermove", (event) => {
      if (!isDown) return;
      const delta = event.clientX - startX;
      if (!isDragging && Math.abs(delta) > 6) {
        isDragging = true;
      }
      if (!isDragging) return;
      container.scrollLeft = startScrollLeft - delta;
    });

    const stopDrag = (event) => {
      if (!isDown) return;
      isDown = false;
      isDragging = false;
      container.style.cursor = "grab";
    };

    container.addEventListener("pointerup", stopDrag);
    container.addEventListener("pointercancel", stopDrag);
    container.addEventListener("pointerleave", stopDrag);
  });
}

function setupRegionSelectors() {
  const provinceSelect = $("province-select");
  const citySelect = $("city-select");
  const districtSelect = $("district-select");
  if (!provinceSelect || !citySelect || !districtSelect) return;
  fillSelect(provinceSelect, Object.keys(regionOptions), "请选择省");
  provinceSelect.addEventListener("change", () => {
    const cities = Object.keys(regionOptions[provinceSelect.value] || {});
    citySelect.disabled = cities.length === 0;
    districtSelect.disabled = true;
    fillSelect(citySelect, cities, cities.length ? "请选择市" : "请先选择省");
    fillSelect(districtSelect, [], "请先选择市");
  });
  citySelect.addEventListener("change", () => {
    const districts = regionOptions[provinceSelect.value]?.[citySelect.value] || [];
    districtSelect.disabled = districts.length === 0;
    fillSelect(districtSelect, districts, districts.length ? "请选择区" : "请先选择市");
  });
}

function fillCustomerFormRegion(province, city, district) {
  const provinceSelect = $("province-select");
  const citySelect = $("city-select");
  const districtSelect = $("district-select");
  if (!provinceSelect || !citySelect || !districtSelect) return;
  provinceSelect.value = province || "";
  const cities = Object.keys(regionOptions[provinceSelect.value] || {});
  citySelect.disabled = cities.length === 0;
  fillSelect(citySelect, cities, cities.length ? "请选择市" : "请先选择省");
  citySelect.value = city || "";
  const districts = regionOptions[provinceSelect.value]?.[citySelect.value] || [];
  districtSelect.disabled = districts.length === 0;
  fillSelect(districtSelect, districts, districts.length ? "请选择区" : "请先选择市");
  districtSelect.value = district || "";
}

function resetCustomerFormMode() {
  state.editingCustomerId = null;
  $("customer-form-title").textContent = "新建客户";
  $("customer-form-submit").textContent = "保存客户";
}

function openModal() {
  $("customer-form").reset();
  $("modal").classList.remove("hidden");
  resetCustomerFormMode();
  $("creator-input").value = currentUser;
  const ownerInput = $("customer-form")?.elements?.owner;
  if (ownerInput) ownerInput.value = currentUser;
  $("created-at-preview").value = "保存后自动生成";
  $("updated-at-preview").value = "保存后自动生成";
  fillCustomerFormRegion("", "", "");
}

function closeModal() {
  $("modal").classList.add("hidden");
  resetCustomerFormMode();
}

function openChangePasswordModal() {
  $("change-password-form")?.reset();
  ["change-password-old", "change-password-new", "change-password-confirm"].forEach((id) => {
    if ($(id)) $(id).type = "password";
  });
  if ($("change-password-toggle")) $("change-password-toggle").checked = false;
  $("change-password-modal")?.classList.remove("hidden");
}

function closeChangePasswordModal() {
  $("change-password-form")?.reset();
  $("change-password-modal")?.classList.add("hidden");
}

function openCreateUserModal() {
  $("account-create-form")?.reset();
  if ($("account-create-password")) $("account-create-password").type = "password";
  if ($("account-create-password-toggle")) $("account-create-password-toggle").checked = false;
  $("account-create-modal")?.classList.remove("hidden");
}

function closeCreateUserModal() {
  $("account-create-modal")?.classList.add("hidden");
}

function openAccountEditModal(user) {
  state.editingUserId = user.id;
  $("account-edit-id").value = user.id;
  $("account-edit-username").value = user.username || "";
  $("account-edit-role").value = user.role || "";
  $("account-edit-alias").value = user.alias || "";
  $("account-edit-password").value = "";
  if ($("account-edit-password")) $("account-edit-password").type = "password";
  if ($("account-edit-password-toggle")) $("account-edit-password-toggle").checked = false;
  $("account-edit-modal").classList.remove("hidden");
}

function closeAccountEditModal() {
  state.editingUserId = null;
  $("account-edit-form")?.reset();
  $("account-edit-modal")?.classList.add("hidden");
}

function openContractModal(customerId = 0) {
  $("contract-form")?.reset();
  fillContractCustomerOptions(customerId);
  state.contractProducts = [];
  renderContractProducts();
  if ($("contract-customer-search")) {
    $("contract-customer-search").value = "";
  }
  filterContractCustomerOptions("");
  $("contract-customer-dropdown")?.classList.add("hidden");
  if ($("contract-order-date")) {
    $("contract-order-date").value = toDateInputValue(new Date());
  }
  if ($("contract-file-list")) {
    $("contract-file-list").textContent = "暂未选择附件";
    $("contract-file-list").classList.remove("has-files");
  }
  $("contract-modal")?.classList.remove("hidden");
}

function closeContractModal() {
  $("contract-modal")?.classList.add("hidden");
}

async function loadUsers() {
  if (currentRole !== "admin") return;
  const users = await api("/api/users");
  state.users = Array.isArray(users) ? users : [];
  renderUsers();
}

async function load() {
  try {
    const [customers, activities, contacts, contracts, payments, summary] = await Promise.all([
      api("/api/customers"),
      api("/api/activities"),
      api("/api/contacts"),
      api("/api/contracts"),
      api("/api/payments"),
      api("/api/dashboard/summary"),
    ]);
    state.customers = Array.isArray(customers) ? customers : [];
    state.activities = Array.isArray(activities) ? activities : [];
    state.contacts = Array.isArray(contacts) ? contacts : [];
    state.contracts = Array.isArray(contracts) ? contracts : [];
    state.payments = Array.isArray(payments) ? payments : [];
    fillCustomerStageFilter();
    renderOverviewMetrics(summary || {});
    renderCustomers();
    renderActivities();
    renderContracts();
    renderPaymentApprovals();
    await loadUsers();
  } catch (err) {
    console.error(err);
  }
}

function filterCustomers(stage = "", keyword = "", customerStage = state.customerFilter.customerStage || "") {
  state.customerFilter = { stage, keyword, customerStage };
  state.customerPage = 1;
  renderCustomers();
}

if ($("user-name")) $("user-name").textContent = currentAlias;
if ($("user-role-label")) $("user-role-label").textContent = currentRole === "admin" ? "管理员" : "普通账户";
if (currentRole === "admin") {
  $("accounts-nav")?.classList.remove("hidden");
}
$("payment-approvals-nav")?.classList.remove("hidden");

$("logout-btn")?.addEventListener("click", () => {
  localStorage.removeItem("crm_token");
  localStorage.removeItem("crm_user");
  localStorage.removeItem("crm_alias");
  localStorage.removeItem("crm_role");
  location.replace("/login.html");
});

bindPasswordToggle("account-create-password-toggle", "account-create-password");
bindPasswordToggle("account-edit-password-toggle", "account-edit-password");
const changePasswordToggle = $("change-password-toggle");
if (changePasswordToggle) {
  changePasswordToggle.addEventListener("change", () => {
    ["change-password-old", "change-password-new", "change-password-confirm"].forEach((id) => {
      if ($(id)) $(id).type = changePasswordToggle.checked ? "text" : "password";
    });
  });
}

document.querySelectorAll(".tab").forEach((tab) => tab.addEventListener("click", () => {
  document.querySelectorAll(".tab").forEach((item) => item.classList.remove("active"));
  tab.classList.add("active");
  filterCustomers(tab.dataset.stage || "", $("search")?.value || "", $("customer-stage-filter")?.value || "");
}));

$("customer-list-2")?.addEventListener("click", (event) => {
  const button = event.target.closest?.("[data-open-detail]");
  if (!button) return;
  const id = Number(button.getAttribute("data-open-detail"));
  if (Number.isFinite(id) && id > 0) openDetailModal(id);
});

document.querySelectorAll(".detail-tab").forEach((tab) => tab.addEventListener("click", () => renderDetailTabs(tab.dataset.tab)));

["search", "search-2"].forEach((id) => {
  $(id)?.addEventListener("input", (event) => {
    const keyword = event.target.value;
    if (id === "search" && $("search-2")) $("search-2").value = keyword;
    if (id === "search-2" && $("search")) $("search").value = keyword;
    filterCustomers(document.querySelector(".tab.active")?.dataset.stage || "", keyword, $("customer-stage-filter")?.value || "");
  });
});

$("customer-stage-filter")?.addEventListener("change", (event) => {
  filterCustomers(
    document.querySelector(".tab.active")?.dataset.stage || "",
    $("search-2")?.value || $("search")?.value || "",
    event.target.value
  );
});

$("pool-search")?.addEventListener("input", renderPoolCustomers);
$("todo-search")?.addEventListener("input", renderTodoCustomers);
$("add-btn-2")?.addEventListener("click", openModal);
$("close-modal")?.addEventListener("click", closeModal);
$("cancel-modal")?.addEventListener("click", closeModal);
$("open-create-user")?.addEventListener("click", openCreateUserModal);
$("close-account-create-modal")?.addEventListener("click", closeCreateUserModal);
$("cancel-account-create-modal")?.addEventListener("click", closeCreateUserModal);
$("close-account-edit-modal")?.addEventListener("click", closeAccountEditModal);
$("cancel-account-edit-modal")?.addEventListener("click", closeAccountEditModal);
$("change-password-btn")?.addEventListener("click", openChangePasswordModal);
$("close-change-password-modal")?.addEventListener("click", closeChangePasswordModal);
$("cancel-change-password-modal")?.addEventListener("click", closeChangePasswordModal);
$("open-create-contract")?.addEventListener("click", openContractModal);
$("open-create-payment")?.addEventListener("click", () => openPaymentModal());
$("close-contract-modal")?.addEventListener("click", closeContractModal);
$("contract-cancel-btn")?.addEventListener("click", closeContractModal);
$("contract-save-draft")?.addEventListener("click", closeContractModal);
$("close-payment-modal")?.addEventListener("click", closePaymentModal);
$("cancel-payment-modal")?.addEventListener("click", closePaymentModal);
$("payment-customer-trigger")?.addEventListener("click", () => {
  $("payment-customer-dropdown")?.classList.toggle("hidden");
  $("payment-customer-search")?.focus();
});
$("payment-customer-search")?.addEventListener("input", (event) => {
  filterPaymentCustomerOptions(event.target.value);
});
$("contract-customer-trigger")?.addEventListener("click", () => {
  $("contract-customer-dropdown")?.classList.toggle("hidden");
  $("contract-customer-search")?.focus();
});
$("contract-customer-search")?.addEventListener("input", (event) => {
  filterContractCustomerOptions(event.target.value);
});
$("contract-attachments")?.addEventListener("change", (event) => {
  const files = Array.from(event.target.files || []);
  const box = $("contract-file-list");
  if (!box) return;
  if (!files.length) {
    box.textContent = "暂未选择附件";
    box.classList.remove("has-files");
    return;
  }
  box.classList.add("has-files");
  box.innerHTML = files.map((file) => `<span class="contract-file-chip">${escapeHtml(file.name)}</span>`).join("");
});
$("detail-close")?.addEventListener("click", closeDetailModal);
$("detail-followup-submit")?.addEventListener("click", submitDetailFollowup);
$("detail-contact-submit")?.addEventListener("click", submitDetailContact);
$("detail-transfer-pool")?.addEventListener("click", handleTransferToPool);
$("detail-basic-edit")?.addEventListener("click", openDetailBasicEdit);
$("detail-basic-cancel")?.addEventListener("click", cancelDetailBasicEdit);
$("detail-basic-save")?.addEventListener("click", saveDetailBasicEdit);

$("customer-prev")?.addEventListener("click", () => {
  if (state.customerPage > 1) {
    state.customerPage -= 1;
    renderCustomerTable(getFilteredCustomers());
  }
});

$("customer-next")?.addEventListener("click", () => {
  const list = getFilteredCustomers();
  const totalPages = Math.max(1, Math.ceil(list.length / state.customerPageSize));
  if (state.customerPage < totalPages) {
    state.customerPage += 1;
    renderCustomerTable(list);
  }
});

$("customer-page-go")?.addEventListener("click", () => {
  const list = getFilteredCustomers();
  const totalPages = Math.max(1, Math.ceil(list.length / state.customerPageSize));
  const target = Number.parseInt($("customer-page-input").value, 10);
  if (!Number.isFinite(target)) return;
  state.customerPage = Math.min(Math.max(target, 1), totalPages);
  renderCustomerTable(list);
});

$("customer-page-input")?.addEventListener("keydown", (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    $("customer-page-go").click();
  }
});

$("more-select")?.addEventListener("change", async (event) => {
  const action = event.target.value;
  if (action === "import") $("import-file").click();
  if (action === "export") {
    try {
      await exportCustomers();
    } catch (err) {
      alert(err.message);
    }
  }
  event.target.value = "";
});

$("import-file")?.addEventListener("change", async (event) => {
  const file = event.target.files?.[0];
  if (!file) return;
  showImportProgressModal();
  try {
    const result = await importCustomers(file, setImportProgress);
    setImportProgress(100, "导入完成");
    if ($("import-progress-summary")) {
      $("import-progress-summary").classList.remove("hidden");
      $("import-progress-summary").textContent = `成功 ${result.imported} 条，跳过 ${result.skipped} 条`;
    }
    renderImportErrors(result.errors || []);
    event.target.value = "";
    await load();
    if (!result.errors || !result.errors.length) {
      window.setTimeout(hideImportProgressModal, 1200);
    }
  } catch (err) {
    setImportProgress(100, "导入失败");
    if ($("import-progress-summary")) {
      $("import-progress-summary").classList.remove("hidden");
      $("import-progress-summary").textContent = err.message;
    }
    renderImportErrors([]);
  }
});

$("close-import-progress-modal")?.addEventListener("click", hideImportProgressModal);
$("import-progress-confirm")?.addEventListener("click", hideImportProgressModal);
$("import-progress-modal")?.addEventListener("click", (event) => {
  if (event.target.id === "import-progress-modal") hideImportProgressModal();
});

$("customer-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  const data = Object.fromEntries(new FormData(event.target).entries());
  if (!String(data.name || "").trim()) {
    alert("客户名称不能为空");
    return;
  }
  if (!String(data.next_contact || "").trim()) {
    alert("下次联系时间不能为空");
    return;
  }
  if (data.next_contact) {
    data.next_contact = toAsiaShanghaiDateTime(data.next_contact, "09:00:00");
  }
  try {
    if (state.editingCustomerId) {
      const updated = await api(`/api/customers/${state.editingCustomerId}`, { method: "PUT", body: JSON.stringify(data) });
      const index = state.customers.findIndex((item) => Number(item.id) === Number(updated.id));
      if (index >= 0) state.customers[index] = { ...state.customers[index], ...updated };
    } else {
      await api("/api/customers", { method: "POST", body: JSON.stringify(data) });
    }
    closeModal();
    await load();
  } catch (err) {
    alert(err.message);
  }
});

$("account-create-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  const payload = Object.fromEntries(new FormData(event.target).entries());
  const passwordError = validateAccountPassword(payload.password);
  if (passwordError) {
    alert(passwordError);
    return;
  }
  try {
    await createUser(payload);
    closeCreateUserModal();
    await loadUsers();
    alert("账户创建成功");
  } catch (err) {
    alert(err.message);
  }
});

$("account-edit-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!state.editingUserId) return;
  const payload = Object.fromEntries(new FormData(event.target).entries());
  if (payload.password) {
    const passwordError = validateAccountPassword(payload.password);
    if (passwordError) {
      alert(passwordError);
      return;
    }
  }
  try {
    await updateUser(state.editingUserId, { alias: payload.alias, password: payload.password });
    closeAccountEditModal();
    await loadUsers();
    alert("账户更新成功");
  } catch (err) {
    alert(err.message);
  }
});

$("change-password-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  const payload = Object.fromEntries(new FormData(event.target).entries());
  const passwordError = validateAccountPassword(payload.new_password);
  if (passwordError) {
    alert(passwordError);
    return;
  }
  if (payload.new_password !== payload.confirm_password) {
    alert("两次输入的新密码不一致");
    return;
  }
  try {
    await changeOwnPassword(payload);
    closeChangePasswordModal();
    alert("密码修改成功，请牢记新密码");
  } catch (err) {
    alert(err.message);
  }
});

$("contract-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  const payload = new FormData(event.target);
  payload.set("products", JSON.stringify((state.contractProducts || []).map(normalizeContractProduct).filter((item) => item.name || item.category || item.unit || item.standard_price || item.sale_price || item.quantity)));
  try {
    await createContract(payload);
    closeContractModal();
    await load();
    alert("合同已提交，等待管理员审批");
  } catch (err) {
    alert(err.message);
  }
});

$("payment-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  const payload = Object.fromEntries(new FormData(event.target).entries());
  payload.customer_id = Number(payload.customer_id || 0);
  payload.contract_id = Number(payload.contract_id || 0);
  payload.payment_amount = Number(payload.payment_amount || 0);
  const customer = getCustomerById(payload.customer_id);
  if (!customer) {
    alert("请选择客户名称");
    return;
  }
  try {
    await createPayment(payload);
    closePaymentModal();
    await load();
    const refreshed = getCustomerById(customer.id);
    if (refreshed) {
      renderDetailPayments(refreshed);
      renderDetailTabs("payment");
    }
    alert("回款信息已保存");
  } catch (err) {
    alert(err.message);
  }
});

document.addEventListener("click", async (event) => {
  const contractCustomerOption = event.target.closest?.("[data-contract-customer-option]");
  if (contractCustomerOption) {
    const customerId = Number(contractCustomerOption.getAttribute("data-contract-customer-option"));
    $("contract-customer").value = Number.isFinite(customerId) ? String(customerId) : "";
    $("contract-customer-trigger").textContent = contractCustomerOption.getAttribute("data-contract-customer-name") || "请选择客户名称";
    $("contract-customer-dropdown")?.classList.add("hidden");
    return;
  }

  const contractCustomerPicker = $("contract-customer-picker");
  if (contractCustomerPicker && !contractCustomerPicker.contains(event.target)) {
    $("contract-customer-dropdown")?.classList.add("hidden");
  }

  const paymentCustomerOption = event.target.closest?.("[data-payment-customer-option]");
  if (paymentCustomerOption) {
    const customerId = Number(paymentCustomerOption.getAttribute("data-payment-customer-option"));
    $("payment-customer-id").value = Number.isFinite(customerId) ? String(customerId) : "";
    $("payment-customer-trigger").textContent = paymentCustomerOption.getAttribute("data-payment-customer-name") || "请选择客户名称";
    $("payment-customer-dropdown")?.classList.add("hidden");
    fillPaymentContractOptions(customerId);
    return;
  }

  const customerPicker = $("payment-customer-picker");
  if (customerPicker && !customerPicker.contains(event.target)) {
    $("payment-customer-dropdown")?.classList.add("hidden");
  }

  const paymentCreateButton = event.target.closest?.("#detail-payment-create");
  if (paymentCreateButton) {
    openPaymentModal();
    return;
  }

  const contractCreateButton = event.target.closest?.("#detail-contract-create");
  if (contractCreateButton) {
    openContractModal(state.selectedCustomerId || 0);
    return;
  }

  const paymentReviewButton = event.target.closest?.("[data-payment-review]");
  if (paymentReviewButton) {
    try {
      await reviewPayment(Number(paymentReviewButton.getAttribute("data-payment-review")), {
        action: paymentReviewButton.getAttribute("data-action"),
      });
      await load();
      const customer = getCustomerById(state.selectedCustomerId);
      if (customer) renderDetailPayments(customer);
    } catch (err) {
      alert(err.message);
    }
    return;
  }

  const contractDetailButton = event.target.closest?.("[data-open-contract]");
  if (contractDetailButton) {
    const id = Number(contractDetailButton.getAttribute("data-open-contract"));
    if (Number.isFinite(id) && id > 0) {
      openContractDetailStandalone(id);
    }
    return;
  }

  const addProductButton = event.target.closest?.("#contract-add-product");
  if (addProductButton) {
    addContractProductRow();
    return;
  }

  const removeProductButton = event.target.closest?.("[data-remove-contract-product]");
  if (removeProductButton) {
    const index = Number(removeProductButton.getAttribute("data-remove-contract-product"));
    state.contractProducts = (state.contractProducts || []).filter((_, itemIndex) => itemIndex !== index);
    renderContractProducts();
    syncContractAmountFromProducts();
    return;
  }

  const detailButton = event.target.closest?.("[data-open-detail]");
  if (detailButton) {
    const id = Number(detailButton.getAttribute("data-open-detail"));
    if (Number.isFinite(id) && id > 0) openDetailModal(id);
    return;
  }

  const editUserButton = event.target.closest?.("[data-edit-user]");
  if (editUserButton) {
    const user = (state.users || []).find((item) => Number(item.id) === Number(editUserButton.getAttribute("data-edit-user")));
    if (user) openAccountEditModal(user);
    return;
  }

  const deleteUserButton = event.target.closest?.("[data-delete-user]");
  if (deleteUserButton) {
    if (!confirm("确定删除这个账户吗？")) return;
    try {
      await api(`/api/users/${deleteUserButton.getAttribute("data-delete-user")}`, { method: "DELETE", body: "{}" });
      await loadUsers();
    } catch (err) {
      alert(err.message);
    }
    return;
  }

  const reviewButton = event.target.closest?.("[data-contract-review]");
  if (reviewButton) {
    try {
      await reviewContract(Number(reviewButton.getAttribute("data-contract-review")), {
        action: reviewButton.getAttribute("data-action"),
      });
      await load();
      const customer = getCustomerById(state.selectedCustomerId);
      if (customer) renderDetailContracts(customer);
    } catch (err) {
      alert(err.message);
    }
  }
});

document.addEventListener("input", (event) => {
  const field = event.target?.getAttribute?.("data-field");
  const row = event.target?.getAttribute?.("data-contract-product");
  if (field == null || row == null) return;
  const index = Number(row);
  if (!Number.isFinite(index) || !state.contractProducts[index]) return;
  const value = ["standard_price", "sale_price", "quantity", "discount"].includes(field)
    ? Number(event.target.value || 0)
    : event.target.value;
  state.contractProducts[index] = {
    ...state.contractProducts[index],
    [field]: value,
  };
  const totalNode = document.querySelector(`[data-contract-product-total="${index}"]`);
  if (totalNode) {
    totalNode.textContent = money(contractProductTotal(state.contractProducts[index]));
  }
  syncContractAmountFromProducts();
});

$("detail-modal")?.addEventListener("click", (event) => {
  if (event.target.id === "detail-modal") closeDetailModal();
});

$("contract-detail-close")?.addEventListener("click", closeContractDetailModal);
$("contract-detail-modal")?.addEventListener("click", (event) => {
  if (event.target.id === "contract-detail-modal") closeContractDetailModal();
});
$("payment-modal")?.addEventListener("click", (event) => {
  if (event.target.id === "payment-modal") closePaymentModal();
});
$("change-password-modal")?.addEventListener("click", (event) => {
  if (event.target.id === "change-password-modal") closeChangePasswordModal();
});

window.addEventListener("hashchange", () => {
  const page = location.hash.replace("#", "") || "overview";
  document.querySelectorAll(".content").forEach((element) => element.classList.toggle("hidden", element.id !== page));
  $("page-title").textContent =
    page === "customers" ? "客户管理" :
    page === "activities" ? "跟进记录" :
    page === "pool" ? "公海池" :
    page === "todos" ? "待办事项" :
    page === "contracts" ? "合同管理" :
    page === "payment-approvals" ? "回款管理" :
    page === "accounts" ? "账户管理" :
    "销售总览";
});

async function bootstrapApp() {
  await loadUiConfig();
  setupRegionSelectors();
  enableHorizontalDragScroll(".table-wrap");
  await load();
  window.dispatchEvent(new Event("hashchange"));
}

bootstrapApp();

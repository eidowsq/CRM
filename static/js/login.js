const form = document.getElementById('login-form');

form.addEventListener('submit', async (e) => {
  e.preventDefault();
  const payload = Object.fromEntries(new FormData(form).entries());
  const res = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  const body = await res.json();
  if (!res.ok) {
    alert(body.error || '登录失败');
    return;
  }
  localStorage.setItem('crm_token', body.data.token);
  localStorage.setItem('crm_user', body.data.user);
  localStorage.setItem('crm_alias', body.data.alias || body.data.user);
  localStorage.setItem('crm_role', body.data.role || 'user');
  location.replace('/');
});

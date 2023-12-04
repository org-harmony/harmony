// redirect to login page if session expired
document.addEventListener('htmx:afterRequest', function(event) {
    if (event.detail.isError) return;

    const request = event.detail.xhr;
    if (!request) return;

    const isLoginPage = request.responseURL.includes('/auth/login');
    if (!isLoginPage) return;

    // TODO would a toast be better?
    window.location.href = '/auth/login';
})
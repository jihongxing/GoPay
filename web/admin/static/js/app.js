// 应用初始化
document.addEventListener('DOMContentLoaded', function() {
    // 高亮当前导航
    highlightCurrentNav();

    // 初始化事件监听
    initEventListeners();
});

// 高亮当前导航
function highlightCurrentNav() {
    const path = window.location.pathname;
    const navLinks = document.querySelectorAll('.nav-link');

    navLinks.forEach(link => {
        link.classList.remove('active');
        const href = link.getAttribute('href');

        if (path === href || (href !== '/admin' && path.startsWith(href))) {
            link.classList.add('active');
        }
    });
}

// 初始化事件监听
function initEventListeners() {
    // ESC 键关闭对话框
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            closeAllDialogs();
        }
    });

    // 点击对话框外部关闭
    document.querySelectorAll('.dialog').forEach(dialog => {
        dialog.addEventListener('click', function(e) {
            if (e.target === dialog) {
                dialog.style.display = 'none';
            }
        });
    });
}

// 关闭所有对话框
function closeAllDialogs() {
    document.querySelectorAll('.dialog').forEach(dialog => {
        dialog.style.display = 'none';
    });
}

// 全局错误处理
window.addEventListener('unhandledrejection', function(event) {
    console.error('Unhandled promise rejection:', event.reason);
    showError('操作失败，请稍后重试');
});

// 页面可见性变化时刷新数据
document.addEventListener('visibilitychange', function() {
    if (!document.hidden) {
        // 页面重新可见时，可以选择刷新数据
        // 这里暂时不自动刷新，避免打断用户操作
    }
});

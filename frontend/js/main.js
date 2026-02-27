// Simple Navigation
function navigateTo(page) {
  window.location.href = page + '.html';
}

// Add Active Class to Nav
document.addEventListener('DOMContentLoaded', () => {
  const navItems = document.querySelectorAll('.nav-item');
  const currentPage = window.location.pathname.split('/').pop().replace('.html', '');

  navItems.forEach(item => {
    if (item.dataset.page === currentPage) {
      item.classList.add('active');
    }
  });
});
// 首页: 轮播图
(function () {
    var banner = document.getElementById('banner');
    if (banner) {
        var slides = Array.prototype.slice.call(banner.querySelectorAll('.banner-slide'));
        var thumbs = Array.prototype.slice.call(banner.querySelectorAll('.banner-thumb'));
        var dots = Array.prototype.slice.call(banner.querySelectorAll('.banner-dot'));
        var current = 0, timer = null;
        var show = function (i) {
            if (slides.length === 0) return;
            current = (i + slides.length) % slides.length;
            [slides, thumbs, dots].forEach(function (list) {
                list.forEach(function (el, j) { el.classList.toggle('active', j === current); });
            });
        };
        var start = function () {
            stop();
            if (slides.length > 1) timer = setInterval(function () { show(current + 1); }, 5000);
        };
        var stop = function () { if (timer) { clearInterval(timer); timer = null; } };
        thumbs.concat(dots).forEach(function (el) {
            el.addEventListener('mouseenter', function () { show(parseInt(el.dataset.index, 10)); });
            el.addEventListener('click', function () { show(parseInt(el.dataset.index, 10)); });
        });
        banner.addEventListener('mouseenter', stop);
        banner.addEventListener('mouseleave', start);
        var x0 = 0;
        banner.addEventListener('touchstart', function (e) { x0 = e.changedTouches[0].pageX; stop(); }, { passive: true });
        banner.addEventListener('touchend', function (e) {
            var d = e.changedTouches[0].pageX - x0;
            if (d <= -50) show(current + 1);
            else if (d >= 50) show(current - 1);
            start();
        });
        start();
    }

})();

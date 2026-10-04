// 影片库: 筛选行默认只显示一行, 放不下时显示「展开 / 收起」(手机版的筛选行是横向滑动, 不处理);
// 筛选区下方的按钮收起 / 展开整个筛选区, 状态记在 localStorage
(function () {
    var panel = document.getElementById('class-panel');
    var panelBtn = document.querySelector('.class-panel-btn');
    var PANEL_KEY = 'gomaccms.classPanel';
    var setPanel = function (open) {
        if (!panel || !panelBtn) return;
        panel.hidden = !open;
        panelBtn.classList.toggle('is-closed', !open);
        panelBtn.setAttribute('aria-expanded', open ? 'true' : 'false');
        panelBtn.title = open ? window.t('filter.hidePanel') : window.t('filter.showPanel');
        panelBtn.firstElementChild.textContent = panelBtn.title;
    };
    if (panelBtn) {
        var stored = null;
        try { stored = localStorage.getItem(PANEL_KEY); } catch (e) {}
        setPanel(stored !== 'closed');
        panelBtn.addEventListener('click', function () {
            var open = panel.hidden;
            setPanel(open);
            try { localStorage.setItem(PANEL_KEY, open ? 'open' : 'closed'); } catch (e) {}
            if (open) setup();
        });
    }

    var rows = document.querySelectorAll('.pianku .module-class-item');
    var desktop = window.matchMedia('(min-width: 560px)');
    var setup = function () {
        Array.prototype.forEach.call(rows, function (row) {
            var box = row.querySelector('.module-item-box');
            var toggle = row.querySelector('.class-toggle');
            if (!box || !toggle) return;
            row.classList.remove('is-collapsed', 'is-expanded');
            var first = box.querySelector('a');
            // 所有选项的高度超过一行才需要展开 / 收起
            var overflow = desktop.matches && first && box.scrollHeight > first.offsetHeight * 1.8;
            toggle.hidden = !overflow;
            if (!overflow) return;
            // 选中的选项在第一行之后时直接展开, 免得看不到当前条件
            var active = box.querySelector('a.active');
            var hidden = active && active.offsetTop - first.offsetTop > first.offsetHeight;
            row.classList.add(hidden ? 'is-expanded' : 'is-collapsed');
            toggle.firstElementChild.textContent = hidden ? window.t('filter.collapse') : window.t('filter.expand');
        });
    };
    Array.prototype.forEach.call(rows, function (row) {
        var toggle = row.querySelector('.class-toggle');
        if (!toggle) return;
        toggle.addEventListener('click', function () {
            var collapsed = row.classList.toggle('is-collapsed');
            row.classList.toggle('is-expanded', !collapsed);
            toggle.firstElementChild.textContent = collapsed ? window.t('filter.expand') : window.t('filter.collapse');
        });
    });
    setup();
    var timer;
    window.addEventListener('resize', function () {
        clearTimeout(timer);
        timer = setTimeout(setup, 150);
    });
})();

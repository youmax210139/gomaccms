// 详情页: 播放源切换、简介展开
(function () {
    var tabs = Array.prototype.slice.call(document.querySelectorAll('.play-source-tabs .tab-item'));
    var lists = Array.prototype.slice.call(document.querySelectorAll('#playlist .tab-list'));
    tabs.forEach(function (tab) {
        tab.addEventListener('click', function () {
            tabs.forEach(function (t) { t.classList.toggle('active', t === tab); });
            lists.forEach(function (l) { l.classList.toggle('active', l.dataset.source === tab.dataset.source); });
        });
    });

    var desc = document.getElementById('detail-desc');
    var toggle = document.getElementById('desc-toggle');
    if (desc && toggle) {
        // 简介没有被截断时不显示「展开全部」
        toggle.hidden = desc.scrollHeight <= desc.clientHeight + 2;
        toggle.addEventListener('click', function () {
            var open = desc.classList.toggle('open');
            toggle.firstElementChild.textContent = open ? window.t('detail.collapse') : window.t('detail.expandAll');
        });
    }
})();

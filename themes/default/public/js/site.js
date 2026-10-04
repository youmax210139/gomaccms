// 全站共用脚本: 顶部搜索 / 导航、观看记录下拉、深色外观、回到顶部、分享、图片加载失败
(function () {
    var $ = function (sel, root) { return (root || document).querySelector(sel); };
    var $$ = function (sel, root) { return Array.prototype.slice.call((root || document).querySelectorAll(sel)); };

    // 轻提示
    window.showToast = function (msg) {
        var el = document.createElement('div');
        el.className = 'site-toast';
        el.textContent = msg;
        document.body.appendChild(el);
        setTimeout(function () { el.classList.add('show'); }, 10);
        setTimeout(function () {
            el.classList.remove('show');
            setTimeout(function () { el.remove(); }, 300);
        }, 1800);
    };

    // 首页导航高亮
    if (location.pathname === '/' || location.pathname === '/index') {
        var home = $('.navbar-item.nav-home');
        if (home) home.classList.add('active');
    }

    // 搜索: 空关键字不提交; 手机版点击搜索图标展开搜索框
    var form = $('#search-form');
    if (form) {
        form.addEventListener('submit', function (e) {
            var input = $('.search-input', form);
            if (!input.value.trim()) {
                e.preventDefault();
                input.focus();
                window.showToast(window.t('search.emptyKeyword'));
            }
        });
    }
    var searchBox = $('.search-box');
    $$('.header-op-search').forEach(function (btn) {
        btn.addEventListener('click', function (e) {
            e.stopPropagation();
            if (!searchBox) return;
            searchBox.classList.toggle('open');
            if (searchBox.classList.contains('open')) $('.search-input', searchBox).focus();
        });
    });
    // 搜索框获得焦点时展开「大家都在搜」(没有热搜词时页面上没有下拉)
    var searchMain = $('.searchbar-main');
    var closeSearch = function () {
        if (searchMain) searchMain.classList.remove('open');
        if (searchBox) searchBox.classList.remove('open');
    };
    if (searchMain && $('.search-recommend', searchMain)) {
        var openSearch = function () { searchMain.classList.add('open'); };
        // 按 Esc 关闭后输入框仍有焦点, 再次点击时也要展开
        ['focus', 'click'].forEach(function (ev) { $('.search-input', searchMain).addEventListener(ev, openSearch); });
        document.addEventListener('keydown', function (e) { if (e.key === 'Escape') closeSearch(); });
    }
    var cancel = $('.cancel-btn');
    if (cancel) cancel.addEventListener('click', closeSearch);

    // 滚动: 顶部导航加背景
    var sidebar = $('.sidebar');
    var onScroll = function () {
        var y = document.documentElement.scrollTop || document.body.scrollTop;
        if (sidebar) sidebar.classList.toggle('sidebar-bg', y > 20);
        document.body.classList.toggle('scrolled', y > 300);
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    onScroll();

    // 观看记录: cookie 中以影片ID为键的 JSON (播放页写入, 保存 30 天); 内容损坏时视为空
    var historyKey = 'filmHistory';
    var getCookie = function (name) {
        var hit = document.cookie.split('; ').filter(function (c) { return c.split('=')[0] === name; })[0];
        return hit ? decodeURIComponent(hit.split('=')[1] || '') : '';
    };
    window.filmHistory = {
        map: function () {
            try {
                return JSON.parse(getCookie(historyKey) || '{}') || {};
            } catch (e) {
                return {};
            }
        },
        // 按观看时间从新到旧
        list: function () {
            var map = this.map();
            return Object.keys(map).map(function (k) { return map[k]; })
                .sort(function (a, b) { return b.timeStamp - a.timeStamp; });
        },
        save: function (map) {
            var expires = new Date(Date.now() + 30 * 24 * 3600 * 1000).toUTCString();
            document.cookie = historyKey + '=' + encodeURIComponent(JSON.stringify(map)) + '; expires=' + expires + '; path=/';
        },
        remove: function (id) {
            var map = this.map();
            delete map[id];
            this.save(map);
        },
        clear: function () { document.cookie = historyKey + "=''; expires=" + new Date(0).toUTCString(); }
    };
    // 只允许站内相对路径 ("//host" 与 "/\host" 会被浏览器当成外部地址)
    window.safeRelativeLink = function (url) {
        return typeof url === 'string' && url.charAt(0) === '/' && url.charAt(1) !== '/' && url.charAt(1) !== '\\' ? url : '#';
    };

    var renderDropHistory = function () {
        $$('.historical').forEach(function (ul) {
            $$('.drop-item:not(.drop-item-title)', ul).forEach(function (li) { li.remove(); });
            var items = window.filmHistory.list().slice(0, 10);
            if (items.length === 0) {
                var empty = document.createElement('li');
                empty.className = 'drop-item drop-item-empty';
                empty.textContent = window.t('history.empty');
                ul.appendChild(empty);
                return;
            }
            items.forEach(function (h) {
                var li = document.createElement('li');
                li.className = 'drop-item';
                var a = document.createElement('a');
                a.className = 'drop-item-link';
                a.href = window.safeRelativeLink(h.link);
                var name = document.createElement('span');
                name.textContent = h.name;
                var ep = document.createElement('cite');
                ep.textContent = h.episode;
                a.appendChild(name);
                a.appendChild(ep);
                li.appendChild(a);
                ul.appendChild(li);
            });
        });
    };
    $$('.drop-history-wrap').forEach(function (wrap) {
        wrap.addEventListener('mouseenter', function () { renderDropHistory(); wrap.classList.add('open'); });
        wrap.addEventListener('mouseleave', function () { wrap.classList.remove('open'); });
        $('.header-op-history', wrap).addEventListener('click', function (e) {
            e.stopPropagation();
            renderDropHistory();
            wrap.classList.toggle('open');
        });
    });
    // 语言切换 (页头下拉, 手机版在页脚): 选择记在 cookie lang (服务端只接受当前方案启用的语言), 重新整理后生效
    $$('.drop-lang-wrap').forEach(function (wrap) {
        $('.header-op-lang', wrap).addEventListener('click', function (e) {
            e.stopPropagation();
            wrap.classList.toggle('open');
        });
    });
    $$('.lang-option').forEach(function (a) {
        a.addEventListener('click', function (e) {
            e.preventDefault();
            document.cookie = 'lang=' + encodeURIComponent(a.dataset.lang) + '; path=/; max-age=31536000; samesite=lax';
            location.reload();
        });
    });
    document.addEventListener('click', function (e) {
        if (!e.target.closest('.drop-lang-wrap')) $$('.drop-lang-wrap').forEach(function (w) { w.classList.remove('open'); });
        if (!e.target.closest('.drop-history-wrap')) $$('.drop-history-wrap').forEach(function (w) { w.classList.remove('open'); });
        if (!e.target.closest('.search-box')) closeSearch();
    });
    $$('.history-clear').forEach(function (btn) {
        btn.addEventListener('click', function (e) {
            e.preventDefault();
            e.stopPropagation();
            window.filmHistory.clear();
            renderDropHistory();
            document.dispatchEvent(new Event('history:cleared'));
        });
    });

    // 深色外观 (存在 localStorage, 页面绘制前已在 layout 中套用)
    var toggle = $('.theme-toggle');
    if (toggle) {
        toggle.addEventListener('click', function () {
            var dark = document.documentElement.classList.toggle('theme-dark');
            try { localStorage.setItem('theme', dark ? 'dark' : 'light'); } catch (e) {}
        });
    }

    // 回到顶部
    var top = $('.retop');
    if (top) top.addEventListener('click', function () { window.scrollTo({ top: 0, behavior: 'smooth' }); });

    // 分享: 复制当前页面地址
    $$('.share-btn').forEach(function (btn) {
        btn.addEventListener('click', function () {
            var text = window.t('share.text', btn.dataset.share || document.title) + location.href;
            var done = function () { window.showToast(window.t('share.copied')); };
            if (navigator.clipboard && window.isSecureContext) {
                navigator.clipboard.writeText(text).then(done, function () { window.prompt(window.t('share.prompt'), text); });
            } else {
                window.prompt(window.t('share.prompt'), text);
            }
        });
    });

    // 海报加载失败时显示默认图片
    document.addEventListener('error', function (e) {
        var t = e.target;
        if (t && t.tagName === 'IMG' && t.classList.contains('card-img') && !t.dataset.fallback) {
            t.dataset.fallback = '1';
            t.src = window.ThemeAssetPath + '/images/no-poster.png';
        }
    }, true);

    // 选集排序 (详情页 / 播放页): 反转每个播放源的选集顺序
    $$('.sort-btn').forEach(function (btn) {
        btn.addEventListener('click', function () {
            $$('.module-play-list-content').forEach(function (box) {
                $$('.module-play-list-link', box).reverse().forEach(function (a) { box.appendChild(a); });
            });
            btn.classList.toggle('active');
        });
    });
})();

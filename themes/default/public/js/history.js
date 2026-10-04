// 观看记录页: 读取播放页写入的 cookie, 可单条删除或清空
(function () {
    var list = document.getElementById('history-page-list');
    var empty = document.getElementById('history-page-empty');
    if (!list || !empty) return;

    var safeImage = function (url) {
        return typeof url === 'string' && (/^https?:\/\//i.test(url) || window.safeRelativeLink(url) !== '#') ? url : '';
    };

    function render() {
        var items = window.filmHistory.list();
        list.innerHTML = '';
        empty.hidden = items.length > 0;
        items.forEach(function (h) {
            var link = window.safeRelativeLink(h.link);
            var card = document.createElement('div');
            card.className = 'module-card-item module-item history-card';

            var poster = document.createElement('a');
            poster.className = 'module-card-item-poster';
            poster.href = link;
            poster.innerHTML = '<div class="module-item-cover"><div class="module-item-note"></div><div class="module-item-pic"><img class="card-img" alt=""></div></div>';
            poster.querySelector('.module-item-note').textContent = h.episode || '';
            var img = poster.querySelector('img');
            img.src = safeImage(h.picture) || (window.ThemeAssetPath + '/images/no-poster.png');
            img.alt = h.name || '';
            img.referrerPolicy = 'no-referrer';

            var info = document.createElement('div');
            info.className = 'module-card-item-info';
            var title = document.createElement('div');
            title.className = 'module-card-item-title';
            var ta = document.createElement('a');
            ta.href = link;
            var strong = document.createElement('strong');
            strong.textContent = h.name || '';
            ta.appendChild(strong);
            title.appendChild(ta);
            info.appendChild(title);
            [[window.t('history.watchedTo'), (h.episode || '') + '  ' + (h.progress || '')], [window.t('history.time'), h.time || '']].forEach(function (row) {
                var item = document.createElement('div');
                item.className = 'module-info-item';
                var c = document.createElement('div');
                c.className = 'module-info-item-content';
                c.textContent = row[0] + ': ' + row[1];
                item.appendChild(c);
                info.appendChild(item);
            });

            var footer = document.createElement('div');
            footer.className = 'module-card-item-footer';
            var play = document.createElement('a');
            play.className = 'play-btn icon-btn';
            play.href = link;
            play.innerHTML = '<i class="icon-play"></i><span>' + window.t('history.continue') + '</span>';
            var del = document.createElement('a');
            del.className = 'play-btn-o';
            del.href = 'javascript:;';
            del.innerHTML = '<span>' + window.t('common.delete') + '</span>';
            del.addEventListener('click', function () {
                window.filmHistory.remove(h.id);
                render();
            });
            footer.appendChild(play);
            footer.appendChild(del);

            card.appendChild(poster);
            card.appendChild(info);
            card.appendChild(footer);
            list.appendChild(card);
        });
    }

    document.addEventListener('history:cleared', render);
    render();
})();

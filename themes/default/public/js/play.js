// 播放页: 播放源 / 选集切换、自动连播、观看记录、xgplayer 初始化与选集面板插件
(function () {
    var sources = JSON.parse(document.getElementById('play-data').textContent);
    var init = window.PLAY_INIT;

    var state = {
        sourceId: init.currentSource,
        episodeIndex: init.currentEpisode,
        autoplay: true,
        // 实际正在播放的选集 (只在 playChange 时更新); 仅切换播放源页签不会改变它, 观看记录以它为准
        playingEpisode: null
    };

    function findSource(id) {
        for (var i = 0; i < sources.length; i++) {
            if (sources[i].id === id) return sources[i];
        }
        return null;
    }

    function currentLinkList() {
        var s = findSource(state.sourceId);
        return s ? s.linkList : [];
    }

    function hasNext() {
        var list = currentLinkList();
        return state.episodeIndex < list.length - 1;
    }

    function updateNextButtonVisibility() {
        var btn = document.getElementById('next-episode-btn');
        if (btn) btn.classList.toggle('disabled', !hasNext());
    }

    function updateActiveHighlights() {
        var tabs = document.querySelectorAll('#play-tab-group .tab-item');
        for (var i = 0; i < tabs.length; i++) {
            tabs[i].classList.toggle('active', tabs[i].dataset.source === state.sourceId);
        }
        var lists = document.querySelectorAll('#play-list .tab-list');
        for (var j = 0; j < lists.length; j++) {
            lists[j].classList.toggle('active', lists[j].dataset.source === state.sourceId);
        }
        var playing = state.playingEpisode || {};
        var links = document.querySelectorAll('#play-list .module-play-list-link');
        for (var k = 0; k < links.length; k++) {
            var active = links[k].dataset.source === playing.sourceId && parseInt(links[k].dataset.episode, 10) === playing.episodeIndex;
            links[k].classList.toggle('active', active);
        }
    }

    function updateEpisodeLabel() {
        var label = document.getElementById('current-episode-label');
        var item = currentLinkList()[state.episodeIndex];
        if (label && item) label.textContent = item.episode;
    }

    // 切换到指定播放源的指定选集并开始播放
    function playChange(sourceId, episodeIndex) {
        var s = findSource(sourceId);
        if (!s || !s.linkList[episodeIndex]) return;
        state.sourceId = sourceId;
        state.episodeIndex = episodeIndex;
        state.playingEpisode = {
            sourceId: sourceId,
            episodeIndex: episodeIndex,
            episode: s.linkList[episodeIndex].episode,
            link: s.linkList[episodeIndex].link
        };
        updateActiveHighlights();
        updateEpisodeLabel();
        updateNextButtonVisibility();
        if (window.history && history.replaceState) {
            history.replaceState(null, '', '/play?id=' + init.mid + '&source=' + encodeURIComponent(sourceId) + '&episode=' + episodeIndex);
        }
        if (window.mPlayer) {
            window.mPlayer.pause();
            window.mPlayer.currentTime = 0;
            window.mPlayer.switchURL(s.linkList[episodeIndex].link);
        }
        if (window.__customPlayList && window.__customPlayList.renderListItems) {
            window.__customPlayList.renderListItems();
        }
    }

    // 下一集 (关闭自动连播时不切换)
    function playNext() {
        if (!hasNext()) return;
        if (state.autoplay) {
            playChange(state.sourceId, state.episodeIndex + 1);
        }
    }

    // 观看记录: 离开页面时写入 cookie (history.js / 顶部下拉读取)
    function pad(n) {
        return n < 10 ? '0' + n : '' + n;
    }

    function secondToTime(seconds) {
        seconds = Math.floor(seconds || 0);
        var hours = Math.floor(seconds / 3600);
        var minutes = Math.floor((seconds % 3600) / 60);
        var secs = seconds % 60;
        return pad(hours) + ':' + pad(minutes) + ':' + pad(secs);
    }

    function dateFormat(timestamp) {
        var d = new Date(timestamp);
        return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + ' ' +
            pad(d.getHours()) + ':' + pad(d.getMinutes()) + ':' + pad(d.getSeconds());
    }

    function saveFilmHistory() {
        if (!window.mPlayer) return;
        var item = state.playingEpisode;
        if (!item || !item.link) return;
        var history = window.filmHistory.map();
        var timeStamp = new Date().getTime();
        var link = '/play?id=' + init.mid + '&source=' + item.sourceId + '&episode=' + item.episodeIndex + '&currentTime=' + window.mPlayer.currentTime;
        history[init.mid] = {
            id: init.mid,
            name: init.name,
            picture: init.picture,
            episode: item.episode,
            time: dateFormat(timeStamp),
            timeStamp: timeStamp,
            source: item.sourceId,
            link: link,
            currentTime: window.mPlayer.currentTime,
            duration: window.mPlayer.duration,
            progress: secondToTime(window.mPlayer.currentTime) + ' / ' + secondToTime(window.mPlayer.duration),
            // 是否为手机 (以 768px 视窗宽度判断)
            devices: window.matchMedia('(max-width: 768px)').matches
        };
        window.filmHistory.save(history);
    }
    window.addEventListener('beforeunload', saveFilmHistory);

    // 自动播放开关
    var autoplayBtn = document.getElementById('autoplay-toggle');
    if (autoplayBtn) {
        autoplayBtn.addEventListener('click', function () {
            state.autoplay = !state.autoplay;
            autoplayBtn.classList.toggle('active', state.autoplay);
            window.showToast(state.autoplay ? window.t('play.autoplayOn') : window.t('play.autoplayOff'));
        });
    }

    // 下一集按钮
    var nextBtn = document.getElementById('next-episode-btn');
    if (nextBtn) {
        nextBtn.addEventListener('click', playNext);
    }

    // 播放源页签: 只切换显示的选集列表, 不影响正在播放的选集 (点击选集时才切换播放)
    var tabGroup = document.getElementById('play-tab-group');
    if (tabGroup) {
        tabGroup.addEventListener('click', function (e) {
            var tab = e.target.closest('.tab-item');
            if (!tab) return;
            state.sourceId = tab.dataset.source;
            updateActiveHighlights();
            updateNextButtonVisibility();
        });
    }

    // 选集点击
    var playList = document.getElementById('play-list');
    if (playList) {
        playList.addEventListener('click', function (e) {
            var link = e.target.closest('.module-play-list-link');
            if (!link) return;
            e.preventDefault();
            playChange(link.dataset.source, parseInt(link.dataset.episode, 10));
        });
    }

    // 以服务端给出的初始播放源 / 选集作为正在播放的选集
    (function seedPlayingEpisode() {
        var item = currentLinkList()[state.episodeIndex];
        if (item) {
            state.playingEpisode = {
                sourceId: state.sourceId,
                episodeIndex: state.episodeIndex,
                episode: item.episode,
                link: item.link
            };
        }
    })();
    updateActiveHighlights();
    updateNextButtonVisibility();

    // xgplayer 的 UMD 版本: Plugin / Events 挂在 Player 上; xgplayer-hls 的插件类即全局 HlsPlayer
    var Player = window.Player;
    var Events = Player.Events;
    var Plugin = Player.Plugin;
    var HlsPlugin = window.HlsPlayer;
    var POSITIONS = Plugin.POSITIONS;

    // 播放器控制栏右侧的「选集」面板. 必须用 ES6 class 继承 (xgplayer 的 Plugin 是 Babel 编译的类, ES5 写法无法正确初始化)
    class PlayListPlugin extends Plugin {
        static get pluginName() {
            return 'customPlayList';
        }

        static get defaultConfig() {
            return {position: POSITIONS.CONTROLS_RIGHT};
        }

        renderListItems() {
            if (!this.listContainer) return;
            this.listContainer.innerHTML = '';
            this.listContainer.className = 'playListContainer';
            var list = currentLinkList();
            var self = this;
            list.forEach(function (item, index) {
                var el = document.createElement('div');
                el.className = index === state.episodeIndex ? 'playlist-item active' : 'playlist-item';
                el.textContent = item.episode;
                el.addEventListener('click', function (e) {
                    e.stopPropagation();
                    playChange(state.sourceId, index);
                    self.renderListItems();
                    self.toggleList();
                });
                el.addEventListener('wheel', function (e) {
                    e.preventDefault();
                    if (self.listContainer) self.listContainer.scrollTop += e.deltaY;
                });
                self.listContainer.appendChild(el);
            });
        }

        toggleList() {
            if (!this.listContainer) return;
            var isHidden = this.listContainer.style.display === 'none' || this.listContainer.style.display === '';
            this.listContainer.style.display = isHidden ? 'block' : 'none';
        }

        afterPlayerInit() {
            var self = this;
            this.bind('click', function (e) {
                e.stopPropagation();
                self.toggleList();
            });
            if (this.player.root) {
                this.player.root.addEventListener('click', function () {
                    if (self.listContainer) self.listContainer.style.display = 'none';
                });
            }
            this.listContainer = document.querySelector('.playList-panel');
            if (this.listContainer) {
                this.listContainer.addEventListener('mouseout', function (e) {
                    e.stopPropagation();
                    self.toggleList();
                });
            }
            this.renderListItems();
            window.__customPlayList = this;
        }

        afterCreate() {
            var self = this;
            this.on([Events.LOADED_DATA], function () {
                self.renderListItems();
            });
        }

        destroy() {
            this.listContainer = null;
        }

        render() {
            return '<xg-icon class="xg-playlist-btn">' + window.t('play.episodes') + '<div class="playList-panel"></div></xg-icon>';
        }
    }

    // 播放器初始化
    var initialLink = currentLinkList()[state.episodeIndex];
    var urls = currentLinkList().map(function (item) {
        return item.link;
    });

    var mPlayer = new Player({
        el: document.getElementById('player-container'),
        url: initialLink ? initialLink.link : init.currentLink,
        poster: window.ThemeAssetPath + '/images/poster-default.jpg',
        width: '100%',
        height: '100%',
        fluid: true,
        videoFillMode: 'contain',
        autoplay: false,
        lang: 'zh-cn',
        volume: 0.7,
        playbackRate: [3, 2, 1.5, 1, 0.75, 0.5],
        playnext: {urlList: urls},
        playsinline: true,
        miniprogress: true,
        'x5-video-orientation': 'landscape',
        'x5-video-player-fullscreen': 'true',
        plugins: [HlsPlugin, PlayListPlugin],
        hls: {
            retryCount: 3,
            retryDelay: 1000,
            loadTimeout: 10000,
            fetchOptions: {mode: 'cors'},
            targetLatency: 10,
            maxLatency: 20,
            preloadTime: 100,
            disconnectTime: 0
        },
        controls: {autoHide: true},
        keyboard: {playbackRate: 3},
        mobile: {
            rotateFullscreen: true,
            hideDefaultControls: true,
            gestureX: true,
            gestureY: true,
            scopeR: 0.15,
            pressRate: 3,
            disablePress: false
        }
    });
    window.mPlayer = mPlayer;

    mPlayer.on(Events.READY, function () {
        // 从观看记录进入时 (?currentTime=) 续播
        var resumeTime = parseFloat(new URLSearchParams(location.search).get('currentTime'));
        if (!isNaN(resumeTime)) {
            mPlayer.currentTime = resumeTime;
        }
        var playNextPlugin = mPlayer.getPlugin('playNext');
        if (playNextPlugin) {
            playNextPlugin.nextHandler = playNext;
        }
    });
    mPlayer.on(Events.PLAY, function () {
        var playBtn = mPlayer.root.querySelector('.xgplayer-start');
        if (playBtn) playBtn.style.display = 'none';
    });
    mPlayer.on(Events.ENDED, function () {
        if (state.autoplay) playNext();
    });
})();

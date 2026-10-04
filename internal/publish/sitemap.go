package publish

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// 生成的 sitemap 中 URL 以 baseToken 开头, 对外提供时替换为请求的 协议://域名 (多域名共用同一分类方案的 sitemap)
const baseToken = "{{BASE_URL}}"

// sitemapName 可对外提供的文件名: 索引、页面、影片分片
var sitemapName = regexp.MustCompile(`^sitemap(-page|-vod-[1-9][0-9]*)?\.xml$`)

// SitemapEntry sitemap 中的一个 URL: 站内路径与内容更新时间 (零值时不输出 lastmod)
type SitemapEntry struct {
	Path    string
	LastMod time.Time
}

// SitemapMeta 一个分类方案最近一次生成的结果 (meta.json)
type SitemapMeta struct {
	VodURLs  int       `json:"vodUrls"`
	PageURLs int       `json:"pageUrls"`
	Shards   int       `json:"shards"`
	BuiltAt  time.Time `json:"builtAt"`
}

// xmlEscape 转义 XML 文本
func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xmlEscapeTo(&b, s)
	return b.String()
}

func xmlEscapeTo(w io.Writer, s string) error {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	_, err := r.WriteString(w, s)
	return err
}

// urlsetWriter 写一个 urlset 文件到 <name>.tmp, close 时补上结尾
type urlsetWriter struct {
	f   *os.File
	w   *bufio.Writer
	tmp string
	n   int
}

func newURLSet(dir, name string) (*urlsetWriter, error) {
	tmp := filepath.Join(dir, name+".tmp")
	f, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}
	u := &urlsetWriter{f: f, w: bufio.NewWriter(f), tmp: tmp}
	_, err = u.w.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	return u, err
}

func (u *urlsetWriter) add(e SitemapEntry) error {
	u.n++
	u.w.WriteString("<url><loc>" + baseToken)
	xmlEscapeTo(u.w, e.Path)
	u.w.WriteString("</loc>")
	if !e.LastMod.IsZero() {
		u.w.WriteString("<lastmod>" + e.LastMod.Format(time.RFC3339) + "</lastmod>")
	}
	_, err := u.w.WriteString("</url>\n")
	return err
}

func (u *urlsetWriter) close() error {
	u.w.WriteString("</urlset>\n")
	if err := u.w.Flush(); err != nil {
		u.f.Close()
		return err
	}
	return u.f.Close()
}

// writeSitemaps 生成一个分类方案的 sitemap 到 dir:
//   - pages 写入 sitemap-page.xml (首页、分类页等)
//   - nextVods 逐批返回影片 (返回空时结束), 每 chunk 个 URL 一个分片 sitemap-vod-N.xml
//   - sitemap.xml 为索引
//
// 全部先写成 .tmp, 都成功后才改名发布 (分片先、索引最后), 搜索引擎不会看到写到一半的文件;
// 失败时删除临时文件, 已发布的 sitemap 保持不变. 新索引发布后删除多余的旧分片
func writeSitemaps(dir string, chunk int, pages []SitemapEntry, nextVods func() ([]SitemapEntry, error), progress func(int)) (SitemapMeta, error) {
	meta := SitemapMeta{BuiltAt: time.Now()}
	if chunk <= 0 {
		chunk = 50000
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return meta, err
	}
	var written []*urlsetWriter
	names := map[*urlsetWriter]string{}
	cleanup := func() {
		for _, u := range written {
			u.f.Close()
			os.Remove(u.tmp)
		}
	}
	open := func(name string) (*urlsetWriter, error) {
		u, err := newURLSet(dir, name)
		if err == nil {
			written = append(written, u)
			names[u] = name
		}
		return u, err
	}
	fail := func(err error) (SitemapMeta, error) {
		cleanup()
		return meta, err
	}

	pageSet, err := open("sitemap-page.xml")
	if err != nil {
		return fail(err)
	}
	for _, p := range pages {
		if err := pageSet.add(p); err != nil {
			return fail(err)
		}
	}
	meta.PageURLs = len(pages)
	if err := pageSet.close(); err != nil {
		return fail(err)
	}

	var shard *urlsetWriter
	for {
		batch, err := nextVods()
		if err != nil {
			return fail(err)
		}
		if len(batch) == 0 {
			break
		}
		for _, e := range batch {
			if shard == nil || shard.n >= chunk {
				if shard != nil {
					if err := shard.close(); err != nil {
						return fail(err)
					}
				}
				meta.Shards++
				if shard, err = open(fmt.Sprintf("sitemap-vod-%d.xml", meta.Shards)); err != nil {
					return fail(err)
				}
			}
			if err := shard.add(e); err != nil {
				return fail(err)
			}
			meta.VodURLs++
		}
		if progress != nil {
			progress(len(batch))
		}
	}
	if shard != nil {
		if err := shard.close(); err != nil {
			return fail(err)
		}
	}

	// 索引
	idxName := "sitemap.xml"
	var idx bytes.Buffer
	idx.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	lastmod := meta.BuiltAt.Format(time.RFC3339)
	for _, u := range written {
		idx.WriteString("<sitemap><loc>" + baseToken + "/" + names[u] + "</loc><lastmod>" + lastmod + "</lastmod></sitemap>\n")
	}
	idx.WriteString("</sitemapindex>\n")
	if err := os.WriteFile(filepath.Join(dir, idxName+".tmp"), idx.Bytes(), 0o644); err != nil {
		os.Remove(filepath.Join(dir, idxName+".tmp"))
		return fail(err)
	}

	// 发布: 分片先改名, 索引最后
	for _, u := range written {
		if err := os.Rename(u.tmp, filepath.Join(dir, names[u])); err != nil {
			os.Remove(filepath.Join(dir, idxName+".tmp"))
			return fail(err)
		}
	}
	if err := os.Rename(filepath.Join(dir, idxName+".tmp"), filepath.Join(dir, idxName)); err != nil {
		return meta, err
	}
	removeStaleShards(dir, meta.Shards)
	if b, err := json.Marshal(meta); err == nil {
		if os.WriteFile(filepath.Join(dir, "meta.json.tmp"), b, 0o644) == nil {
			os.Rename(filepath.Join(dir, "meta.json.tmp"), filepath.Join(dir, "meta.json"))
		}
	}
	return meta, nil
}

// removeStaleShards 删除编号大于 keep 的旧影片分片
func removeStaleShards(dir string, keep int) {
	files, _ := filepath.Glob(filepath.Join(dir, "sitemap-vod-*.xml"))
	for _, f := range files {
		var n int
		if _, err := fmt.Sscanf(filepath.Base(f), "sitemap-vod-%d.xml", &n); err == nil && n > keep {
			os.Remove(f)
		}
	}
}

// readSitemapMeta 分类方案最近一次生成的结果 (没有生成过时为零值)
func readSitemapMeta(dir string) SitemapMeta {
	var m SitemapMeta
	if b, err := os.ReadFile(filepath.Join(dir, "meta.json")); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

// ServeSitemapFile 输出 dir 中的 sitemap 文件, 并把 URL 前缀替换为 base (协议://域名);
// 文件名不合法或文件不存在时返回 os.ErrNotExist
func ServeSitemapFile(w io.Writer, dir, name, base string) error {
	if !sitemapName.MatchString(name) {
		return os.ErrNotExist
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer f.Close()
	esc := xmlEscape(base)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	bw := bufio.NewWriter(w)
	for sc.Scan() {
		bw.WriteString(strings.Replace(sc.Text(), baseToken, esc, 1))
		bw.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return bw.Flush()
}

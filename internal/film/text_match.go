package film

import (
	"strings"

	"gomaccms/internal/db"

	"gorm.io/gorm/clause"
)

// likeEscaper 转义 LIKE 的通配符 (MySQL / MariaDB 默认以反斜线转义)
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// textMatch 文字搜索条件。MySQL 用 ngram 全文索引 (MATCH ... AGAINST);
// MariaDB 没有 ngram, 中文无法分词, 改用 LIKE '%词%' (结果正确, 但用不到索引)。
// phrase 为 true 时整个 word 是一个词组 (BOOLEAN MODE "word"); 为 false 时 word 以空格分隔, 任一词匹配即可 (自然语言模式)。
func textMatch(word string, phrase bool, columns ...string) clause.Expr {
	if !db.IsMariaDB {
		cols := strings.Join(columns, ", ")
		if phrase {
			return clause.Expr{SQL: "MATCH(" + cols + ") AGAINST(? IN BOOLEAN MODE)", Vars: []any{`"` + word + `"`}}
		}
		return clause.Expr{SQL: "MATCH(" + cols + ") AGAINST(?)", Vars: []any{word}}
	}
	terms := []string{word}
	if !phrase {
		terms = strings.Fields(word)
	}
	var conds []string
	var vars []any
	for _, term := range terms {
		for _, col := range columns {
			conds = append(conds, col+" LIKE ?")
			vars = append(vars, "%"+likeEscaper.Replace(term)+"%")
		}
	}
	if len(conds) == 0 {
		return clause.Expr{SQL: "1 = 0"}
	}
	return clause.Expr{SQL: "(" + strings.Join(conds, " OR ") + ")", Vars: vars}
}

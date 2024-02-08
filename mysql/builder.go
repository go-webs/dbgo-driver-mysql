package mysql

import (
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-webs/dbgo"
	"reflect"
	"strings"
)

const DriverName = "mysql"

type Builder struct {
	//prefix string
}

func init() {
	dbgo.Register(DriverName, &Builder{})
}

func (b Builder) ToSql(c *dbgo.Context) (sql4prepare string, binds []any, err error) {
	selects, anies := b.ToSqlSelect(c)
	table, binds2, err := b.ToSqlTable(c)
	if err != nil {
		return sql4prepare, binds2, err
	}
	joins, binds3, err := b.ToSqlJoin(c)
	if err != nil {
		return sql4prepare, binds3, err
	}
	wheres, binds4, err := b.ToSqlWhere(c)
	if err != nil {
		return sql4prepare, binds4, err
	}
	orderBy := b.ToSqlOrderBy(c)
	limit, binds5 := b.ToSqlLimitOffset(c)
	groupBys := b.ToSqlGroupBy(c)
	havings, binds6, err := b.ToSqlHaving(c)

	binds = append(binds, anies...)
	binds = append(binds, binds2...)
	binds = append(binds, binds3...)
	binds = append(binds, binds4...)
	binds = append(binds, binds5...)
	binds = append(binds, binds6...)

	var locking string
	if c.Locking != nil {
		if *c.Locking {
			locking = "FOR UPDATE"
		} else {
			locking = "LOCK IN SHARE MODE"
		}
	}

	sql4prepare = NamedSprintf("SELECT :selects FROM :table :join :wheres :groupBys :havings :orderBy :pagination :locking", selects, table, joins, wheres, groupBys, havings, orderBy, limit, locking)
	return
}

func (b Builder) ToSqlIncDec(c *dbgo.Context, symbol string, data map[string]any) (sql4prepare string, values []any, err error) {
	prepare, anies, err := b.ToSqlTable(c)
	if err != nil {
		return sql4prepare, values, err
	}
	where, val, err := b.ToSqlWhere(c)
	if err != nil {
		return sql4prepare, values, err
	}
	values = append(values, anies...)
	values = append(values, val...)

	var tmp []string
	for k, v := range data {
		tmp = append(tmp, fmt.Sprintf("%s=%s%s?", BackQuotes(k), BackQuotes(k), symbol))
		values = append(values, v)
	}
	sql4prepare = fmt.Sprintf("UPDATE %s SET %s %s", prepare, strings.Join(tmp, ","), where)
	return
}

func (Builder) ToSqlSelect(c *dbgo.Context) (sql4prepare string, binds []any) {
	var cols []string
	for _, col := range c.SelectClause.Columns {
		if col.IsRaw {
			cols = append(cols, col.Name)
			binds = append(binds, col.Binds...)
		} else {
			if col.Alias == "" {
				cols = append(cols, BackQuotes(col.Name))
			} else {
				cols = append(cols, fmt.Sprintf("%s AS %s", BackQuotes(col.Name), col.Alias))
			}
		}
	}
	var distinct string
	if c.SelectClause.Distinct {
		distinct = "DISTINCT "
	}
	sql4prepare = fmt.Sprintf("%s%s", distinct, strings.Join(cols, ", "))
	return
}

func (b Builder) ToSqlTable(c *dbgo.Context) (sql4prepare string, binds []any, err error) {
	return b.buildSqlTable(c.TableClause, c.Prefix)
}

func (b Builder) buildSqlTable(tab dbgo.TableClause, prefix string) (sql4prepare string, binds []any, err error) {
	if v, ok := tab.Tables.(dbgo.IBuilder); ok {
		return v.ToSql()
	}
	rfv := reflect.Indirect(reflect.ValueOf(tab.Tables))
	switch rfv.Kind() {
	case reflect.String:
		sql4prepare = BackQuotes(fmt.Sprintf("%s%s", prefix, tab.Tables))
	case reflect.Struct:
		sql4prepare = b.buildTableName(rfv.Type(), prefix)
	case reflect.Slice:
		if rfv.Type().Elem().Kind() == reflect.Struct {
			sql4prepare = b.buildTableName(rfv.Type().Elem(), prefix)
		} else {
			err = errors.New("table param must be string or struct(slice) bind with 1 or 2 params")
			return
		}
	default:
		err = errors.New("table must string | struct | slice")
		return
	}
	return strings.TrimSpace(fmt.Sprintf("%s %s", sql4prepare, tab.Alias)), binds, err
}

func (b Builder) toSqlWhere(wc dbgo.WhereClause) (sql4prepare string, binds []any, err error) {
	if len(wc.Conditions) == 0 {
		return
	}
	var sql4prepareArr []string
	for _, v := range wc.Conditions {
		switch v.(type) {
		case dbgo.TypeWhereRaw:
			item := v.(dbgo.TypeWhereRaw)
			sql4prepareArr = append(sql4prepareArr, fmt.Sprintf("%s %s", item.LogicalOp, item.Column))
			binds = append(binds, item.Bindings...)
		case dbgo.TypeWhereNested:
			item := v.(dbgo.TypeWhereNested)
			var tmp = dbgo.Context{}
			item.Column(&tmp.WhereClause)
			prepare, anies, err := b.ToSqlWhere(&tmp)
			if err != nil {
				return sql4prepare, binds, err
			}
			sql4prepareArr = append(sql4prepareArr, fmt.Sprintf("%s (%s)", item.LogicalOp, prepare))
			binds = append(binds, anies...)
		case dbgo.TypeWhereSubQuery:
			item := v.(dbgo.TypeWhereSubQuery)
			query, anies, err := item.SubQuery.ToSql()
			if err != nil {
				return sql4prepare, binds, err
			}
			sql4prepareArr = append(sql4prepareArr, fmt.Sprintf("%s %s %s (%s)", item.LogicalOp, BackQuotes(item.Column), item.Operator, query))
			binds = append(binds, anies...)
		case dbgo.TypeWhereStandard:
			item := v.(dbgo.TypeWhereStandard)
			sql4prepareArr = append(sql4prepareArr, fmt.Sprintf("%s %s %s ?", item.LogicalOp, BackQuotes(item.Column), item.Operator))
			binds = append(binds, item.Value)
		case dbgo.TypeWhereIn:
			item := v.(dbgo.TypeWhereIn)
			values := ToSlice(item.Value)
			sql4prepareArr = append(sql4prepareArr, fmt.Sprintf("%s %s %s (%s)", item.LogicalOp, BackQuotes(item.Column), item.Operator, strings.Repeat("?,", len(values)-1)+"?"))
			binds = append(binds, values...)
		case dbgo.TypeWhereBetween:
			item := v.(dbgo.TypeWhereBetween)
			values := ToSlice(item.Value)
			sql4prepareArr = append(sql4prepareArr, fmt.Sprintf("%s %s %s ? AND ?", item.LogicalOp, BackQuotes(item.Column), item.Operator))
			binds = append(binds, values...)
		}
	}
	if len(sql4prepareArr) > 0 {
		sql4prepare = strings.TrimSpace(strings.Trim(strings.Trim(strings.TrimSpace(strings.Join(sql4prepareArr, " ")), "AND"), "OR"))
	}
	return
}
func (b Builder) ToSqlWhere(c *dbgo.Context) (sql4prepare string, binds []any, err error) {
	sql4prepare, binds, err = b.toSqlWhere(c.WhereClause)
	if sql4prepare != "" {
		if c.WhereClause.Not {
			sql4prepare = fmt.Sprintf("NOT %s", sql4prepare)
		}
		sql4prepare = fmt.Sprintf("WHERE %s", sql4prepare)
	}
	return
}

func (b Builder) ToSqlJoin(c *dbgo.Context) (sql4prepare string, binds []any, err error) {
	if c.JoinClause.Err != nil {
		return sql4prepare, binds, c.JoinClause.Err
	}
	if len(c.JoinClause.JoinItems) == 0 {
		return
	}
	var prepare string
	for _, v := range c.JoinClause.JoinItems {
		switch item := v.(type) {
		case dbgo.TypeJoinStandard:
			prepare, binds, err = b.buildSqlTable(item.TableClause, c.Prefix)
			if err != nil {
				return
			}
			sql4prepare = fmt.Sprintf("%s JOIN %s ON %s %s %s", item.Type, prepare, BackQuotes(item.Column1), item.Operator, BackQuotes(item.Column2))
		case dbgo.TypeJoinSub:
			sql4prepare, binds, err = item.IBuilder.ToSql()
			if err != nil {
				return
			}
		case dbgo.TypeJoinOn:
			var tjo dbgo.TypeJoinOnCondition
			item.OnClause(&tjo)
			if len(tjo.Conditions) == 0 {
				return
			}
			var sqlArr []string
			for _, cond := range tjo.Conditions {
				sqlArr = append(sqlArr, fmt.Sprintf("%s %s %s %s", cond.Relation, BackQuotes(cond.Column1), cond.Operator, BackQuotes(cond.Column2)))
			}

			sql4prepare = TrimPrefixAndOr(strings.Join(sqlArr, " "))
		}
	}
	return
}

func (b Builder) ToSqlGroupBy(c *dbgo.Context) (sql4prepare string) {
	if len(c.Groups) > 0 {
		var tmp []string
		for _, col := range c.Groups {
			if col.IsRaw {
				tmp = append(tmp, col.Column)
			} else {
				tmp = append(tmp, BackQuotes(col.Column))
			}
		}
		sql4prepare = fmt.Sprintf("GROUP BY %s", strings.Join(tmp, ","))
	}
	return
}
func (b Builder) ToSqlHaving(c *dbgo.Context) (sql4prepare string, binds []any, err error) {
	sql4prepare, binds, err = b.toSqlWhere(c.HavingClause.WhereClause)
	if sql4prepare != "" {
		sql4prepare = fmt.Sprintf("HAVING %s", sql4prepare)
	}
	return
}
func (b Builder) ToSqlOrderBy(c *dbgo.Context) (sql4prepare string) {
	if len(c.OrderByClause.Columns) == 0 {
		return
	}
	var orderBys []string
	for _, v := range c.OrderByClause.Columns {
		if v.IsRaw {
			orderBys = append(orderBys, v.Column)
		} else {
			if v.Direction == "" {
				orderBys = append(orderBys, BackQuotes(v.Column))
			} else {
				orderBys = append(orderBys, fmt.Sprintf("%s %s", BackQuotes(v.Column), v.Direction))
			}
		}
	}
	sql4prepare = fmt.Sprintf("ORDER BY %s", strings.Join(orderBys, ", "))
	return
}

func (b Builder) ToSqlLimitOffset(c *dbgo.Context) (sqlSegment string, binds []any) {
	var offset int
	if c.LimitOffsetClause.Offset > 0 {
		offset = c.LimitOffsetClause.Offset
	} else if c.LimitOffsetClause.Page > 0 {
		offset = c.LimitOffsetClause.Limit * (c.LimitOffsetClause.Page - 1)
	}
	if c.LimitOffsetClause.Limit > 0 {
		if offset > 0 {
			sqlSegment = "LIMIT ? OFFSET ?"
			binds = append(binds, c.LimitOffsetClause.Limit, offset)
		} else {
			sqlSegment = "LIMIT ?"
			binds = append(binds, c.LimitOffsetClause.Limit)
		}
	}
	return
}

func (b Builder) ToSqlInsert(c *dbgo.Context, obj any, ignoreCase string, onDuplicateKeys []string, mustFields ...string) (sqlSegment string, binds []any, err error) {
	var ctx = *c
	rfv := reflect.Indirect(reflect.ValueOf(obj))
	switch rfv.Kind() {
	case reflect.Struct:
		var datas []map[string]any
		datas, err = dbgo.StructsToInsert(obj, mustFields...)
		if err != nil {
			return
		}
		ctx.Table(obj)
		return b.toSqlInsert(&ctx, datas, ignoreCase, onDuplicateKeys)
	case reflect.Slice:
		switch rfv.Type().Elem().Kind() {
		case reflect.Struct:
			c.Table(obj)
			var datas []map[string]any
			datas, err = dbgo.StructsToInsert(obj, mustFields...)
			if err != nil {
				return
			}
			return b.toSqlInsert(c, datas, ignoreCase, onDuplicateKeys)
		default:
			return b.toSqlInsert(c, obj, ignoreCase, onDuplicateKeys)
		}
	default:
		return b.toSqlInsert(c, obj, ignoreCase, onDuplicateKeys)
	}
}

func (b Builder) ToSqlUpdate(c *dbgo.Context, obj any, mustFields ...string) (sqlSegment string, binds []any, err error) {
	rfv := reflect.Indirect(reflect.ValueOf(obj))
	switch rfv.Kind() {
	case reflect.Struct:
		dataMap, pk, pkValue, err := dbgo.StructToUpdate(obj, mustFields...)
		if err != nil {
			return sqlSegment, binds, err
		}
		var ctx = *c
		ctx.Table(obj)
		if pk != "" {
			ctx.Where(pk, pkValue)
		}
		return b.toSqlUpdate(&ctx, dataMap)
	default:
		return b.toSqlUpdate(c, obj)
	}
}

func (b Builder) ToSqlDelete(c *dbgo.Context, obj any) (sqlSegment string, binds []any, err error) {
	var ctx = *c
	rfv := reflect.Indirect(reflect.ValueOf(obj))
	switch rfv.Kind() {
	case reflect.Struct:
		data, err := dbgo.StructToDelete(obj)
		if err != nil {
			return sqlSegment, binds, err
		}
		ctx.Table(obj).Where(data)
		return b.toSqlDelete(&ctx)
	case reflect.Int64, reflect.Int32, reflect.String:
		ctx.Where("id", obj)
		return b.toSqlDelete(&ctx)
	default:
		err = errors.New("obj must be struct or id value")
	}
	return
}

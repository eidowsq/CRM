package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	dbMu            sync.RWMutex
	db              *sql.DB
	dbDriver        string
	registeredType  = map[string]reflect.Type{}
	registeredTable = map[string]string{}
	modelOrder      []string
)

func RegisterDataBase(_ string, driver string, dsn string) error {
	handle, err := sql.Open(driver, dsn)
	if err != nil {
		return err
	}
	if err := handle.Ping(); err != nil {
		_ = handle.Close()
		return err
	}
	dbMu.Lock()
	db = handle
	dbDriver = driver
	dbMu.Unlock()
	if driver == "sqlite3" || driver == "sqlite" {
		if _, err := handle.Exec("PRAGMA foreign_keys = ON"); err != nil {
			_ = handle.Close()
			return err
		}
	}
	return nil
}

func RegisterModel(ms ...any) {
	dbMu.Lock()
	defer dbMu.Unlock()
	for _, m := range ms {
		t := reflect.TypeOf(m)
		if t == nil {
			continue
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		key := t.String()
		if _, ok := registeredType[key]; ok {
			continue
		}
		registeredType[key] = t
		registeredTable[key] = toTableName(t.Name())
		modelOrder = append(modelOrder, key)
	}
}

func RunSyncdb(_ string, _ bool, _ bool) error {
	dbMu.RLock()
	handle := db
	dbMu.RUnlock()
	if handle == nil {
		return errors.New("database not registered")
	}
	for _, key := range modelOrder {
		t := registeredType[key]
		stmt := buildCreateTableSQL(t, registeredTable[key])
		if _, err := handle.Exec(stmt); err != nil {
			return fmt.Errorf("syncdb failed for table %s: %w; sql=%s", registeredTable[key], err, stmt)
		}
	}
	return nil
}

type Orm struct{}

func NewOrm() Orm { return Orm{} }

func Exec(query string, args ...any) error {
	handle, err := getDB()
	if err != nil {
		return err
	}
	_, err = handle.Exec(query, args...)
	return err
}

type QuerySeter struct {
	table      string
	modelType  reflect.Type
	filters    []filter
	order      string
	relatedSel bool
}

type filter struct {
	field string
	op    string
	value any
}

func (o Orm) QueryTable(model any) QuerySeter {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	key := t.String()
	return QuerySeter{table: registeredTable[key], modelType: t}
}

func (q QuerySeter) RelatedSel(_ ...string) QuerySeter {
	q.relatedSel = true
	return q
}

func (q QuerySeter) Filter(field string, value any) QuerySeter {
	op := "eq"
	if strings.Contains(field, "__icontains") {
		field = strings.TrimSuffix(field, "__icontains")
		op = "icontains"
	}
	q.filters = append(q.filters, filter{field: field, op: op, value: value})
	return q
}

func (q QuerySeter) OrderBy(field string) QuerySeter {
	q.order = field
	return q
}

func (q QuerySeter) Count() (int64, error) {
	handle, err := getDB()
	if err != nil {
		return 0, err
	}
	sqlStr, args := q.buildSQL("COUNT(*)")
	row := handle.QueryRow(sqlStr, args...)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q QuerySeter) All(dest any) (int64, error) {
	handle, err := getDB()
	if err != nil {
		return 0, err
	}
	sqlStr, args := q.buildSQL("*")
	rows, err := handle.Query(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return 0, errors.New("dest must be pointer to slice")
	}

	slice := rv.Elem()
	elemType := slice.Type().Elem()
	columns, _ := rows.Columns()
	var count int64

	for rows.Next() {
		itemPtr := reflect.New(elemType)
		if err := scanRow(rows, columns, itemPtr.Elem(), q.modelType); err != nil {
			return count, err
		}
		if q.relatedSel {
			if err := loadRelations(itemPtr.Elem()); err != nil {
				return count, err
			}
		}
		slice = reflect.Append(slice, itemPtr.Elem())
		count++
	}
	if err := rows.Err(); err != nil {
		return count, err
	}
	rv.Elem().Set(slice)
	return count, nil
}

func (q QuerySeter) buildSQL(selectExpr string) (string, []any) {
	whereParts := make([]string, 0, len(q.filters))
	args := make([]any, 0, len(q.filters))
	for _, f := range q.filters {
		column := toColumnName(f.field)
		switch f.op {
		case "icontains":
			whereParts = append(whereParts, fmt.Sprintf("%s LIKE ?", column))
			args = append(args, "%"+fmt.Sprint(f.value)+"%")
		default:
			whereParts = append(whereParts, fmt.Sprintf("%s = ?", column))
			args = append(args, f.value)
		}
	}

	sqlStr := fmt.Sprintf("SELECT %s FROM %s", selectExpr, q.table)
	if len(whereParts) > 0 {
		sqlStr += " WHERE " + strings.Join(whereParts, " AND ")
	}
	if q.order != "" {
		desc := strings.HasPrefix(q.order, "-")
		column := toColumnName(strings.TrimPrefix(q.order, "-"))
		if desc {
			sqlStr += " ORDER BY " + column + " DESC"
		} else {
			sqlStr += " ORDER BY " + column + " ASC"
		}
	}
	return sqlStr, args
}

func (o Orm) Insert(obj any) (int64, error) {
	handle, err := getDB()
	if err != nil {
		return 0, err
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return 0, errors.New("insert expects pointer to struct")
	}
	st := v.Elem()
	prepareTimestamps(st, true)

	columns, values, args := buildInsertPayload(st)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableNameOf(st.Type()), strings.Join(columns, ", "), strings.Join(values, ", "))
	res, err := handle.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if f := st.FieldByName("Id"); f.IsValid() && f.CanSet() {
		f.SetInt(id)
	}
	return id, nil
}

func (o Orm) Read(obj any) error {
	handle, err := getDB()
	if err != nil {
		return err
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("read expects pointer to struct")
	}
	st := v.Elem()
	id := st.FieldByName("Id")
	if !id.IsValid() {
		return errors.New("missing Id field")
	}
	sqlStr := fmt.Sprintf("SELECT * FROM %s WHERE id = ? LIMIT 1", tableNameOf(st.Type()))
	rows, err := handle.Query(sqlStr, id.Interface())
	if err != nil {
		return err
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	if !rows.Next() {
		return errors.New("not found")
	}
	if err := scanRow(rows, cols, st, st.Type()); err != nil {
		return err
	}
	return rows.Err()
}

func (o Orm) Update(obj any, _ ...string) (int64, error) {
	handle, err := getDB()
	if err != nil {
		return 0, err
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return 0, errors.New("update expects pointer to struct")
	}
	st := v.Elem()
	prepareTimestamps(st, false)

	setParts, args := buildUpdatePayload(st)
	if len(setParts) == 0 {
		return 0, nil
	}
	args = append(args, st.FieldByName("Id").Interface())
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", tableNameOf(st.Type()), strings.Join(setParts, ", "))
	res, err := handle.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	aff, err := res.RowsAffected()
	return aff, err
}

func (o Orm) Delete(obj any) (int64, error) {
	handle, err := getDB()
	if err != nil {
		return 0, err
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return 0, errors.New("delete expects pointer to struct")
	}
	st := v.Elem()
	id := st.FieldByName("Id")
	res, err := handle.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableNameOf(st.Type())), id.Interface())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func getDB() (*sql.DB, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	if db == nil {
		return nil, errors.New("database not registered")
	}
	return db, nil
}

func tableNameOf(t reflect.Type) string {
	key := t.String()
	if table, ok := registeredTable[key]; ok {
		return table
	}
	if t.Kind() == reflect.Ptr {
		return tableNameOf(t.Elem())
	}
	return toTableName(t.Name())
}

func toTableName(name string) string {
	return strings.ToLower(name)
}

func toColumnName(name string) string {
	var out []rune
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				out = append(out, '_')
			}
			out = append(out, r+'a'-'A')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func buildCreateTableSQL(t reflect.Type, table string) string {
	if currentDriver() == "sqlite3" || currentDriver() == "sqlite" {
		return buildSQLiteCreateTableSQL(t, table)
	}
	switch t.Name() {
	case "Customer":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(120) NOT NULL,
			industry VARCHAR(80) NULL,
			level VARCHAR(20) NOT NULL DEFAULT '普通',
			stage VARCHAR(30) NOT NULL DEFAULT '潜在客户',
			phone VARCHAR(30) NULL,
			mobile VARCHAR(30) NULL,
			website VARCHAR(200) NULL,
			source VARCHAR(60) NULL,
			next_contact DATETIME NULL,
			note LONGTEXT NULL,
			creator VARCHAR(50) NULL,
			owner VARCHAR(50) NULL,
			follow_up_record LONGTEXT NULL,
			province VARCHAR(40) NULL,
			city VARCHAR(40) NULL,
			district VARCHAR(40) NULL,
			address VARCHAR(200) NULL,
			email VARCHAR(120) NULL,
			amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case "Contact":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			customer_id BIGINT NOT NULL,
			name VARCHAR(60) NOT NULL,
			role VARCHAR(60) NULL,
			phone VARCHAR(30) NULL,
			email VARCHAR(120) NULL,
			created_at DATETIME NOT NULL,
			CONSTRAINT fk_` + table + `_customer FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case "Activity":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			customer_id BIGINT NOT NULL,
			type VARCHAR(30) NOT NULL,
			content LONGTEXT NOT NULL,
			next_action VARCHAR(200) NULL,
			created_at DATETIME NOT NULL,
			CONSTRAINT fk_` + table + `_customer FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case "Contract":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			customer_id BIGINT NOT NULL,
			serial_no VARCHAR(80) NULL,
			title VARCHAR(120) NOT NULL,
			business_name VARCHAR(120) NULL,
			amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			order_date DATETIME NULL,
			start_date DATETIME NULL,
			end_date DATETIME NULL,
			customer_signer VARCHAR(80) NULL,
			company_signer VARCHAR(80) NULL,
			content LONGTEXT NULL,
			attachments LONGTEXT NULL,
			products LONGTEXT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			submitter VARCHAR(50) NOT NULL,
			reviewer VARCHAR(50) NULL,
			review_note LONGTEXT NULL,
			created_at DATETIME NOT NULL,
			reviewed_at DATETIME NULL,
			CONSTRAINT fk_` + table + `_customer FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case "Payment":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			customer_id BIGINT NOT NULL,
			contract_id BIGINT NOT NULL,
			serial_no VARCHAR(80) NULL,
			contract_title VARCHAR(120) NULL,
			contract_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			payment_method VARCHAR(40) NULL,
			payment_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			payment_date DATETIME NULL,
			note LONGTEXT NULL,
			created_at DATETIME NOT NULL,
			CONSTRAINT fk_` + table + `_customer FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE,
			CONSTRAINT fk_` + table + `_contract FOREIGN KEY (contract_id) REFERENCES contract(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case "User":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(50) NOT NULL UNIQUE,
			alias VARCHAR(50) NULL,
			password VARCHAR(120) NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'user',
			created_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	default:
		return ""
	}
}

func currentDriver() string {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return dbDriver
}

func buildSQLiteCreateTableSQL(t reflect.Type, table string) string {
	switch t.Name() {
	case "Customer":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			industry TEXT NULL,
			level TEXT NOT NULL DEFAULT '普通',
			stage TEXT NOT NULL DEFAULT '潜在客户',
			phone TEXT NULL,
			mobile TEXT NULL,
			website TEXT NULL,
			source TEXT NULL,
			next_contact DATETIME NULL,
			note TEXT NULL,
			creator TEXT NULL,
			owner TEXT NULL,
			follow_up_record TEXT NULL,
			province TEXT NULL,
			city TEXT NULL,
			district TEXT NULL,
			address TEXT NULL,
			email TEXT NULL,
			amount REAL NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`
	case "Contact":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			role TEXT NULL,
			phone TEXT NULL,
			email TEXT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE
		)`
	case "Activity":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			next_action TEXT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE
		)`
	case "Contract":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER NOT NULL,
			serial_no TEXT NULL,
			title TEXT NOT NULL,
			business_name TEXT NULL,
			amount REAL NOT NULL DEFAULT 0,
			order_date DATETIME NULL,
			start_date DATETIME NULL,
			end_date DATETIME NULL,
			customer_signer TEXT NULL,
			company_signer TEXT NULL,
			content TEXT NULL,
			attachments TEXT NULL,
			products TEXT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			submitter TEXT NOT NULL,
			reviewer TEXT NULL,
			review_note TEXT NULL,
			created_at DATETIME NOT NULL,
			reviewed_at DATETIME NULL,
			FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE
		)`
	case "Payment":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER NOT NULL,
			contract_id INTEGER NOT NULL,
			serial_no TEXT NULL,
			contract_title TEXT NULL,
			contract_amount REAL NOT NULL DEFAULT 0,
			payment_method TEXT NULL,
			payment_amount REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'pending',
			payment_date DATETIME NULL,
			note TEXT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE,
			FOREIGN KEY (contract_id) REFERENCES contract(id) ON DELETE CASCADE
		)`
	case "User":
		return `CREATE TABLE IF NOT EXISTS ` + table + ` (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			alias TEXT NULL,
			password TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			created_at DATETIME NOT NULL
		)`
	default:
		return ""
	}
}

func buildInsertPayload(st reflect.Value) ([]string, []string, []any) {
	var columns, placeholders []string
	var args []any
	for i := 0; i < st.NumField(); i++ {
		sf := st.Type().Field(i)
		fv := st.Field(i)
		if sf.Name == "Id" {
			continue
		}
		if sf.Type.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			if id, ok := relationIDFromPtr(fv); ok {
				columns = append(columns, toColumnName(sf.Name)+"_id")
				placeholders = append(placeholders, "?")
				args = append(args, id)
			}
			continue
		}
		if sf.Name == "CreatedAt" || sf.Name == "UpdatedAt" {
			columns = append(columns, toColumnName(sf.Name))
			placeholders = append(placeholders, "?")
			args = append(args, fv.Interface())
			continue
		}
		columns = append(columns, toColumnName(sf.Name))
		placeholders = append(placeholders, "?")
		args = append(args, fieldValueForSQL(fv))
	}
	return columns, placeholders, args
}

func buildUpdatePayload(st reflect.Value) ([]string, []any) {
	var setParts []string
	var args []any
	for i := 0; i < st.NumField(); i++ {
		sf := st.Type().Field(i)
		fv := st.Field(i)
		if sf.Name == "Id" {
			continue
		}
		if sf.Type.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			if id, ok := relationIDFromPtr(fv); ok {
				setParts = append(setParts, toColumnName(sf.Name)+"_id = ?")
				args = append(args, id)
			}
			continue
		}
		if sf.Name == "CreatedAt" {
			continue
		}
		setParts = append(setParts, toColumnName(sf.Name)+" = ?")
		args = append(args, fieldValueForSQL(fv))
	}
	return setParts, args
}

func fieldValueForSQL(v reflect.Value) any {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Struct:
		if t, ok := v.Interface().(time.Time); ok {
			if t.IsZero() {
				return nil
			}
			return t
		}
	}
	return v.Interface()
}

func relationIDFromPtr(v reflect.Value) (any, bool) {
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil, false
	}
	id := v.Elem().FieldByName("Id")
	if !id.IsValid() {
		return nil, false
	}
	return id.Interface(), true
}

func scanRow(rows *sql.Rows, columns []string, dest reflect.Value, modelType reflect.Type) error {
	meta := buildFieldMeta(dest.Type())
	scanTargets := make([]any, len(columns))
	rawTargets := make([]sql.NullString, len(columns))
	intTargets := make([]sql.NullInt64, len(columns))
	floatTargets := make([]sql.NullFloat64, len(columns))
	timeTargets := make([]sql.NullTime, len(columns))

	for i, col := range columns {
		field := meta[col]
		if !field.valid && strings.HasSuffix(col, "_id") {
			base := strings.TrimSuffix(col, "_id")
			if relField, ok := meta[base]; ok && relField.valid && relField.kind == reflect.Ptr {
				scanTargets[i] = &intTargets[i]
				continue
			}
		}
		switch field.kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			scanTargets[i] = &intTargets[i]
		case reflect.Float32, reflect.Float64:
			scanTargets[i] = &floatTargets[i]
		case reflect.Struct:
			scanTargets[i] = &timeTargets[i]
		default:
			scanTargets[i] = &rawTargets[i]
		}
	}

	if err := rows.Scan(scanTargets...); err != nil {
		return err
	}

	for i, col := range columns {
		field := meta[col]
		if !field.valid {
			continue
		}
		fv := dest.Field(field.index)
		switch field.kind {
		case reflect.String:
			if s, ok := scanTargets[i].(*sql.NullString); ok && s.Valid {
				fv.SetString(s.String)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if n, ok := scanTargets[i].(*sql.NullInt64); ok && n.Valid {
				fv.SetInt(n.Int64)
			}
		case reflect.Float32, reflect.Float64:
			if n, ok := scanTargets[i].(*sql.NullFloat64); ok && n.Valid {
				fv.SetFloat(n.Float64)
			}
		case reflect.Struct:
			if t, ok := scanTargets[i].(*sql.NullTime); ok && t.Valid {
				fv.Set(reflect.ValueOf(t.Time))
			}
		case reflect.Ptr:
			// relation foreign key ids are handled by relation column case below
		}
	}

	for i, col := range columns {
		if strings.HasSuffix(col, "_id") {
			base := strings.TrimSuffix(col, "_id")
			field := meta[base]
			if !field.valid || field.kind != reflect.Ptr {
				continue
			}
			if n, ok := scanTargets[i].(*sql.NullInt64); ok && n.Valid {
				ptr := reflect.New(field.relationType)
				ptr.Elem().FieldByName("Id").SetInt(n.Int64)
				fv := dest.Field(field.index)
				if fv.CanSet() {
					fv.Set(ptr)
				}
			}
		}
	}
	return nil
}

type fieldMeta struct {
	valid        bool
	index        int
	kind         reflect.Kind
	relationType reflect.Type
}

func buildFieldMeta(t reflect.Type) map[string]fieldMeta {
	meta := map[string]fieldMeta{}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		if sf.Type.Kind() == reflect.Ptr && sf.Type.Elem().Kind() == reflect.Struct {
			key := toColumnName(sf.Name)
			meta[key] = fieldMeta{valid: true, index: i, kind: reflect.Ptr, relationType: sf.Type.Elem()}
			continue
		}
		meta[toColumnName(sf.Name)] = fieldMeta{valid: true, index: i, kind: sf.Type.Kind()}
	}
	return meta
}

func loadRelations(dest reflect.Value) error {
	handle, err := getDB()
	if err != nil {
		return err
	}
	for i := 0; i < dest.NumField(); i++ {
		sf := dest.Type().Field(i)
		fv := dest.Field(i)
		if sf.Type.Kind() != reflect.Ptr || fv.IsNil() {
			continue
		}
		idField := fv.Elem().FieldByName("Id")
		if !idField.IsValid() || idField.Int() == 0 {
			continue
		}
		table := tableNameOf(sf.Type.Elem())
		rows, err := handle.Query(fmt.Sprintf("SELECT * FROM %s WHERE id = ? LIMIT 1", table), idField.Interface())
		if err != nil {
			return err
		}
		defer rows.Close()
		columns, _ := rows.Columns()
		if !rows.Next() {
			continue
		}
		ptr := reflect.New(sf.Type.Elem())
		if err := scanRow(rows, columns, ptr.Elem(), sf.Type.Elem()); err != nil {
			return err
		}
		fv.Set(ptr)
	}
	return nil
}

func prepareTimestamps(st reflect.Value, isInsert bool) {
	now := time.Now()
	if f := st.FieldByName("CreatedAt"); f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(time.Time{}) && isInsert && f.IsZero() {
		f.Set(reflect.ValueOf(now))
	}
	if f := st.FieldByName("UpdatedAt"); f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(time.Time{}) {
		if isInsert {
			if f.IsZero() {
				f.Set(reflect.ValueOf(now))
			}
		} else {
			f.Set(reflect.ValueOf(now))
		}
	}
}

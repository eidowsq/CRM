package controllers

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"crm/models"
	"github.com/beego/beego/v2/client/orm"
)

var customerImportHeaders = []string{
	"客户名称", "客户级别", "客户行业", "客户来源", "成交状态", "电话", "网址", "下次联系时间", "备注",
	"手机", "创建人", "更新时间", "创建时间", "负责人", "跟进记录", "省", "市", "区/县", "详细地址",
}

var customerImportAliases = map[string][]string{
	"name":             {"客户名称", "名称", "客户名"},
	"level":            {"客户级别", "级别", "客户等级"},
	"industry":         {"客户行业", "行业"},
	"source":           {"客户来源", "来源"},
	"stage":            {"成交状态", "客户阶段", "状态"},
	"phone":            {"电话", "联系电话", "座机"},
	"website":          {"网址", "网站", "官网"},
	"next_contact":     {"下次联系时间", "下次跟进时间", "下次联系"},
	"note":             {"备注", "备注信息"},
	"mobile":           {"手机", "手机号", "手机号码"},
	"creator":          {"创建人", "录入人"},
	"updated_at":       {"更新时间", "最后更新时间"},
	"created_at":       {"创建时间", "录入时间"},
	"owner":            {"负责人", "归属人", "所属人"},
	"follow_up_record": {"跟进记录", "跟进内容", "跟进备注"},
	"province":         {"省", "省份"},
	"city":             {"市", "城市"},
	"district":         {"区/县", "区县", "区", "县"},
	"address":          {"详细地址", "地址", "联系地址"},
}

func (c *CustomerController) Import() {
	if err := c.Ctx.Request.ParseMultipartForm(10 << 20); err != nil {
		c.error("上传文件解析失败", http.StatusBadRequest)
		return
	}

	file, _, err := c.Ctx.Request.FormFile("file")
	if err != nil {
		c.error("请选择 CSV 文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	rows, err := readCSVRows(file)
	if err != nil {
		c.error("CSV 文件读取失败", http.StatusBadRequest)
		return
	}
	if len(rows) == 0 {
		c.error("CSV 内容为空", http.StatusBadRequest)
		return
	}

	headerIndex, headerMap := detectHeader(rows)
	startRow := 0
	if headerIndex >= 0 {
		startRow = headerIndex + 1
	}

	o := orm.NewOrm()
	imported := 0
	skipped := 0
	importUser := strings.TrimSpace(currentUser(c.Ctx.Request))
	errItems := make([]map[string]any, 0, 20)

	for idx, row := range rows[startRow:] {
		rowNo := startRow + idx + 1
		if isEmptyRow(row) {
			continue
		}

		var customer models.Customer
		if headerIndex >= 0 {
			customer, err = customerFromMappedRow(row, headerMap)
		} else {
			customer, err = customerFromCSVRow(row)
		}
		if err != nil || strings.TrimSpace(customer.Name) == "" {
			skipped++
			if len(errItems) < cap(errItems) {
				reason := "客户名称不能为空"
				if err != nil {
					reason = err.Error()
				}
				errItems = append(errItems, map[string]any{
					"row":     rowNo,
					"name":    strings.TrimSpace(customer.Name),
					"reason":  reason,
					"content": strings.Join(row, " | "),
				})
			}
			continue
		}
		if customer.NextContact.IsZero() {
			customer.NextContact = time.Now().AddDate(0, 0, 7)
		}
		if customer.CreatedAt.IsZero() {
			customer.CreatedAt = time.Now()
		}
		if customer.UpdatedAt.IsZero() {
			customer.UpdatedAt = customer.CreatedAt
		}
		if importUser != "" {
			customer.Creator = importUser
			customer.Owner = importUser
		}
		if _, err := o.Insert(&customer); err != nil {
			skipped++
			if len(errItems) < cap(errItems) {
				errItems = append(errItems, map[string]any{
					"row":     rowNo,
					"name":    customer.Name,
					"reason":  err.Error(),
					"content": strings.Join(row, " | "),
				})
			}
			continue
		}
		if strings.TrimSpace(customer.FollowUpRecord) != "" {
			nextAction := ""
			if !customer.NextContact.IsZero() {
				nextAction = customer.NextContact.Format("2006-01-02")
			}
			activity := models.Activity{
				Customer:   &models.Customer{Id: customer.Id},
				Type:       "跟进记录",
				Content:    customer.FollowUpRecord,
				NextAction: nextAction,
				Publisher:  currentUser(c.Ctx.Request),
				CreatedAt:  customer.UpdatedAt,
			}
			if activity.CreatedAt.IsZero() {
				activity.CreatedAt = customer.CreatedAt
			}
			if _, err := o.Insert(&activity); err != nil && len(errItems) < cap(errItems) {
				errItems = append(errItems, map[string]any{
					"row":    rowNo,
					"name":   customer.Name,
					"reason": "跟进记录同步失败: " + err.Error(),
				})
			}
		}
		imported++
	}

	c.respond(map[string]any{
		"imported": imported,
		"skipped":  skipped,
		"errors":   errItems,
		"columns":  customerImportHeaders,
	}, http.StatusOK)
}

func (c *CustomerController) Export() {
	if currentRole(c.Ctx.Request) != "admin" {
		c.error("仅管理员可以导出客户", http.StatusForbidden)
		return
	}

	var customers []models.Customer
	_, err := orm.NewOrm().QueryTable(new(models.Customer)).OrderBy("-updated_at").All(&customers)
	if err != nil {
		c.error("客户导出失败", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	buf.WriteString("\ufeff")
	writer := csv.NewWriter(&buf)
	_ = writer.Write(customerImportHeaders)
	for _, customer := range customers {
		record := []string{
			customer.Name,
			customer.Level,
			customer.Industry,
			customer.Source,
			exportStage(customer.Stage),
			customer.Phone,
			customer.Website,
			formatExportTime(customer.NextContact),
			customer.Note,
			customer.Mobile,
			customer.Creator,
			formatExportTime(customer.UpdatedAt),
			formatExportTime(customer.CreatedAt),
			customer.Owner,
			customer.FollowUpRecord,
			customer.Province,
			customer.City,
			customer.District,
			customer.Address,
		}
		_ = writer.Write(record)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		c.error("客户导出失败", http.StatusInternalServerError)
		return
	}

	filename := "customers_" + time.Now().Format("20060102_150405") + ".csv"
	c.Ctx.ResponseWriter.Header().Set("Content-Type", "text/csv; charset=utf-8")
	c.Ctx.ResponseWriter.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Ctx.ResponseWriter.WriteHeader(http.StatusOK)
	_, _ = c.Ctx.ResponseWriter.Write(buf.Bytes())
}

func readCSVRows(file io.Reader) ([][]string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	text := strings.ReplaceAll(string(data), "\ufeff", "")

	reader := csv.NewReader(strings.NewReader(text))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err == nil && len(rows) > 0 {
		return normalizeRows(rows), nil
	}

	var fallback [][]string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fallback = append(fallback, strings.Split(line, "\t"))
	}
	if len(fallback) == 0 {
		return nil, errors.New("empty csv")
	}
	return normalizeRows(fallback), nil
}

func normalizeRows(rows [][]string) [][]string {
	for i := range rows {
		for j := range rows[i] {
			rows[i][j] = strings.TrimSpace(strings.ReplaceAll(rows[i][j], "\ufeff", ""))
		}
	}
	return rows
}

func detectHeader(rows [][]string) (int, map[string]int) {
	for i, row := range rows {
		headerMap := buildHeaderMap(row)
		if headerMap["name"] >= 0 {
			return i, headerMap
		}
	}
	return -1, nil
}

func buildHeaderMap(row []string) map[string]int {
	result := map[string]int{}
	for key := range customerImportAliases {
		result[key] = -1
	}
	for idx, col := range row {
		canonical := canonicalHeader(col)
		for key, aliases := range customerImportAliases {
			for _, alias := range aliases {
				if canonical == canonicalHeader(alias) {
					result[key] = idx
					break
				}
			}
		}
	}
	return result
}

func canonicalHeader(value string) string {
	replacer := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "", "/", "", "／", "", "-", "", "_", "")
	return replacer.Replace(strings.TrimSpace(value))
}

func isEmptyRow(row []string) bool {
	for _, item := range row {
		if strings.TrimSpace(item) != "" {
			return false
		}
	}
	return true
}

func customerFromMappedRow(row []string, headerMap map[string]int) (models.Customer, error) {
	get := func(key string) string {
		index, ok := headerMap[key]
		if !ok || index < 0 || index >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[index])
	}

	nextContact, _ := parseImportTime(get("next_contact"))
	updatedAt, _ := parseImportTime(get("updated_at"))
	createdAt, _ := parseImportTime(get("created_at"))

	return models.Customer{
		Name:           get("name"),
		Level:          get("level"),
		Industry:       get("industry"),
		Source:         get("source"),
		Stage:          normalizeStage(get("stage")),
		Phone:          get("phone"),
		Website:        get("website"),
		NextContact:    nextContact,
		Note:           get("note"),
		Mobile:         get("mobile"),
		Creator:        get("creator"),
		UpdatedAt:      updatedAt,
		CreatedAt:      createdAt,
		Owner:          get("owner"),
		FollowUpRecord: get("follow_up_record"),
		Province:       get("province"),
		City:           get("city"),
		District:       get("district"),
		Address:        get("address"),
	}, nil
}

func customerFromCSVRow(row []string) (models.Customer, error) {
	for len(row) < 19 {
		row = append(row, "")
	}
	nextContact, _ := parseImportTime(row[7])
	updatedAt, _ := parseImportTime(row[11])
	createdAt, _ := parseImportTime(row[12])

	return models.Customer{
		Name:           strings.TrimSpace(row[0]),
		Level:          strings.TrimSpace(row[1]),
		Industry:       strings.TrimSpace(row[2]),
		Source:         strings.TrimSpace(row[3]),
		Stage:          normalizeStage(strings.TrimSpace(row[4])),
		Phone:          strings.TrimSpace(row[5]),
		Website:        strings.TrimSpace(row[6]),
		NextContact:    nextContact,
		Note:           strings.TrimSpace(row[8]),
		Mobile:         strings.TrimSpace(row[9]),
		Creator:        strings.TrimSpace(row[10]),
		UpdatedAt:      updatedAt,
		CreatedAt:      createdAt,
		Owner:          strings.TrimSpace(row[13]),
		FollowUpRecord: strings.TrimSpace(row[14]),
		Province:       strings.TrimSpace(row[15]),
		City:           strings.TrimSpace(row[16]),
		District:       strings.TrimSpace(row[17]),
		Address:        strings.TrimSpace(row[18]),
	}, nil
}

func normalizeStage(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "潜在客户"
	}
	if value == "未成交" {
		return "未成交"
	}
	return value
}

func exportStage(value string) string {
	value = strings.TrimSpace(value)
	if value == "潜在客户" {
		return "未成交"
	}
	return value
}

func formatExportTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func parseImportTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"2006.01.02 15:04:05",
		"2006.01.02 15:04",
		"2006.01.02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, nil
		}
	}
	if excelFloat, err := strconv.ParseFloat(value, 64); err == nil {
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local)
		return base.Add(time.Duration(excelFloat*24) * time.Hour), nil
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", value)
}

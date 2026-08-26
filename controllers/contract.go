package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"crm/models"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type ContractController struct{ APIController }

func (c *ContractController) List() {
	user := currentUser(c.Ctx.Request)
	role := currentRole(c.Ctx.Request)
	var items []models.Contract
	_, err := orm.NewOrm().QueryTable(new(models.Contract)).RelatedSel().OrderBy("-created_at").All(&items)
	if err != nil {
		c.error(err.Error(), http.StatusInternalServerError)
		return
	}
	if role != "admin" {
		filtered := make([]models.Contract, 0, len(items))
		for _, item := range items {
			if strings.TrimSpace(item.Submitter) == user {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	c.respond(items, http.StatusOK)
}

func (c *ContractController) Create() {
	var input struct {
		CustomerID int     `json:"customer_id"`
		SerialNo   string  `json:"serial_no"`
		Title      string  `json:"title"`
		BusinessName string `json:"business_name"`
		Amount     float64 `json:"amount"`
		OrderDate  string  `json:"order_date"`
		StartDate  string  `json:"start_date"`
		EndDate    string  `json:"end_date"`
		CustomerSigner string `json:"customer_signer"`
		CompanySigner  string `json:"company_signer"`
		Content    string  `json:"content"`
		Products   string  `json:"products"`
	}
	attachmentsJSON := "[]"
	productsJSON := "[]"

	if strings.Contains(c.Ctx.Request.Header.Get("Content-Type"), "multipart/form-data") {
		if err := c.Ctx.Request.ParseMultipartForm(32 << 20); err != nil {
			c.error("合同表单解析失败", http.StatusBadRequest)
			return
		}
		input.CustomerID, _ = strconv.Atoi(strings.TrimSpace(c.Ctx.Request.FormValue("customer_id")))
		input.SerialNo = strings.TrimSpace(c.Ctx.Request.FormValue("serial_no"))
		input.Title = strings.TrimSpace(c.Ctx.Request.FormValue("title"))
		input.BusinessName = strings.TrimSpace(c.Ctx.Request.FormValue("business_name"))
		input.Amount, _ = strconv.ParseFloat(strings.TrimSpace(c.Ctx.Request.FormValue("amount")), 64)
		input.OrderDate = strings.TrimSpace(c.Ctx.Request.FormValue("order_date"))
		input.StartDate = strings.TrimSpace(c.Ctx.Request.FormValue("start_date"))
		input.EndDate = strings.TrimSpace(c.Ctx.Request.FormValue("end_date"))
		input.CustomerSigner = strings.TrimSpace(c.Ctx.Request.FormValue("customer_signer"))
		input.CompanySigner = strings.TrimSpace(c.Ctx.Request.FormValue("company_signer"))
		input.Content = strings.TrimSpace(c.Ctx.Request.FormValue("content"))
		productsJSON = strings.TrimSpace(c.Ctx.Request.FormValue("products"))
		if productsJSON == "" {
			productsJSON = "[]"
		}

		if c.Ctx.Request.MultipartForm != nil {
			files := c.Ctx.Request.MultipartForm.File["attachments"]
			if len(files) > 0 {
				attachments, err := saveContractAttachments(files)
				if err != nil {
					c.error(err.Error(), http.StatusInternalServerError)
					return
				}
				raw, _ := json.Marshal(attachments)
				attachmentsJSON = string(raw)
			}
		}
	} else if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式不正确", http.StatusBadRequest)
		return
	} else {
		productsJSON = strings.TrimSpace(input.Products)
		if productsJSON == "" {
			productsJSON = "[]"
		}
	}

	if input.CustomerID == 0 || strings.TrimSpace(input.Title) == "" {
		c.error("合同标题和客户不能为空", http.StatusBadRequest)
		return
	}
	orderDate := parseDateValue(input.OrderDate)
	startDate := parseDateValue(input.StartDate)
	endDate := parseDateValue(input.EndDate)
	contract := models.Contract{
		Customer:    &models.Customer{Id: input.CustomerID},
		SerialNo:    strings.TrimSpace(input.SerialNo),
		Title:       strings.TrimSpace(input.Title),
		BusinessName: strings.TrimSpace(input.BusinessName),
		Amount:      input.Amount,
		OrderDate:   orderDate,
		StartDate:   startDate,
		EndDate:     endDate,
		CustomerSigner: strings.TrimSpace(input.CustomerSigner),
		CompanySigner:  strings.TrimSpace(input.CompanySigner),
		Content:     strings.TrimSpace(input.Content),
		Attachments: attachmentsJSON,
		Products:    productsJSON,
		Status:      "pending",
		Submitter:   currentUser(c.Ctx.Request),
		CreatedAt:   time.Now(),
	}
	if _, err := orm.NewOrm().Insert(&contract); err != nil {
		c.error(err.Error(), http.StatusInternalServerError)
		return
	}
	c.respond(contract, http.StatusCreated)
}

func parseDateValue(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		return parsed
	}
	return time.Time{}
}

type contractAttachment struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func saveContractAttachments(files []*multipart.FileHeader) ([]contractAttachment, error) {
	uploadRoot := strings.TrimSpace(web.AppConfig.DefaultString("uploads", "uploads"))
	if uploadRoot == "" {
		uploadRoot = "uploads"
	}
	publicRoot := filepath.ToSlash(strings.Trim(uploadRoot, "/\\"))
	dir := uploadRoot
	if !filepath.IsAbs(dir) {
		dir = filepath.Join("static", dir)
	}
	dir = filepath.Join(dir, "contracts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建附件目录失败")
	}

	attachments := make([]contractAttachment, 0, len(files))
	for _, fileHeader := range files {
		src, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("读取附件失败")
		}

		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), sanitizeUploadName(fileHeader.Filename))
		targetPath := filepath.Join(dir, filename)
		dst, err := os.Create(targetPath)
		if err != nil {
			_ = src.Close()
			return nil, fmt.Errorf("保存附件失败")
		}

		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			_ = src.Close()
			return nil, fmt.Errorf("保存附件失败")
		}
		_ = dst.Close()
		_ = src.Close()

		attachments = append(attachments, contractAttachment{
			Name: fileHeader.Filename,
			URL:  "/" + filepath.ToSlash(filepath.Join(publicRoot, "contracts", filename)),
		})
	}

	return attachments, nil
}

func sanitizeUploadName(name string) string {
	safe := filepath.Base(strings.TrimSpace(name))
	safe = strings.ReplaceAll(safe, "..", "")
	safe = strings.ReplaceAll(safe, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	safe = strings.ReplaceAll(safe, " ", "_")
	if safe == "" {
		return "attachment"
	}
	return safe
}

func (c *ContractController) Review() {
	if currentRole(c.Ctx.Request) != "admin" {
		c.error("只有管理员可以审批合同", http.StatusForbidden)
		return
	}
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id == 0 {
		c.error("合同不存在", http.StatusBadRequest)
		return
	}
	var input struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式不正确", http.StatusBadRequest)
		return
	}
	var contract models.Contract
	contract.Id = id
	if err := orm.NewOrm().Read(&contract); err != nil {
		c.error("合同不存在", http.StatusNotFound)
		return
	}
	switch strings.ToLower(strings.TrimSpace(input.Action)) {
	case "approve":
		contract.Status = "approved"
	case "reject":
		contract.Status = "rejected"
	default:
		c.error("无效的审批动作", http.StatusBadRequest)
		return
	}
	contract.Reviewer = currentUser(c.Ctx.Request)
	contract.ReviewNote = strings.TrimSpace(input.Note)
	contract.ReviewedAt = time.Now()
	if _, err := orm.NewOrm().Update(&contract); err != nil {
		c.error(err.Error(), http.StatusInternalServerError)
		return
	}
	c.respond(contract, http.StatusOK)
}

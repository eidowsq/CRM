package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"crm/models"
	"github.com/beego/beego/v2/client/orm"
)

type PaymentController struct{ APIController }

func (c *PaymentController) List() {
	customerID, _ := strconv.Atoi(c.GetString("customer_id"))
	qs := orm.NewOrm().QueryTable(new(models.Payment)).RelatedSel()
	if customerID > 0 {
		qs = qs.Filter("customer_id", customerID)
	}
	var items []models.Payment
	_, err := qs.OrderBy("-payment_date").All(&items)
	if err != nil {
		c.error(err.Error(), http.StatusInternalServerError)
		return
	}
	c.respond(items, http.StatusOK)
}

func (c *PaymentController) Create() {
	var input struct {
		CustomerID    int     `json:"customer_id"`
		ContractID    int     `json:"contract_id"`
		SerialNo      string  `json:"serial_no"`
		PaymentDate   string  `json:"payment_date"`
		PaymentMethod string  `json:"payment_method"`
		PaymentAmount float64 `json:"payment_amount"`
		Note          string  `json:"note"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式不正确", http.StatusBadRequest)
		return
	}
	if input.CustomerID == 0 || input.ContractID == 0 || strings.TrimSpace(input.SerialNo) == "" {
		c.error("回款编号、客户名称、合同编号不能为空", http.StatusBadRequest)
		return
	}

	var contract models.Contract
	contract.Id = input.ContractID
	if err := orm.NewOrm().Read(&contract); err != nil {
		c.error("合同不存在", http.StatusBadRequest)
		return
	}

	paymentDate := time.Now()
	if strings.TrimSpace(input.PaymentDate) != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(input.PaymentDate), time.Local); err == nil {
			paymentDate = parsed
		}
	}

	payment := models.Payment{
		Customer:       &models.Customer{Id: input.CustomerID},
		Contract:       &models.Contract{Id: input.ContractID},
		SerialNo:       strings.TrimSpace(input.SerialNo),
		ContractTitle:  contract.Title,
		ContractAmount: contract.Amount,
		PaymentMethod:  strings.TrimSpace(input.PaymentMethod),
		PaymentAmount:  input.PaymentAmount,
		Status:         "pending",
		PaymentDate:    paymentDate,
		Note:           strings.TrimSpace(input.Note),
		CreatedAt:      time.Now(),
	}
	if _, err := orm.NewOrm().Insert(&payment); err != nil {
		c.error(err.Error(), http.StatusInternalServerError)
		return
	}
	c.respond(payment, http.StatusCreated)
}

func (c *PaymentController) Review() {
	if currentRole(c.Ctx.Request) != "admin" {
		c.error("只有管理员可以审批回款", http.StatusForbidden)
		return
	}

	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id == 0 {
		c.error("回款记录不存在", http.StatusBadRequest)
		return
	}

	var input struct {
		Action string `json:"action"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式不正确", http.StatusBadRequest)
		return
	}

	var payment models.Payment
	payment.Id = id
	if err := orm.NewOrm().Read(&payment); err != nil {
		c.error("回款记录不存在", http.StatusNotFound)
		return
	}

	switch strings.ToLower(strings.TrimSpace(input.Action)) {
	case "approve":
		payment.Status = "approved"
	case "reject":
		payment.Status = "rejected"
	default:
		c.error("无效的审批动作", http.StatusBadRequest)
		return
	}

	if _, err := orm.NewOrm().Update(&payment); err != nil {
		c.error(err.Error(), http.StatusInternalServerError)
		return
	}
	c.respond(payment, http.StatusOK)
}

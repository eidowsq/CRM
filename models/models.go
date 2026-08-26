package models

import (
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type Customer struct {
	Id             int       `orm:"auto" json:"id"`
	Name           string    `orm:"size(120)" json:"name"`
	Industry       string    `orm:"size(80);null" json:"industry"`
	Level          string    `orm:"size(20);default(普通)" json:"level"`
	Stage          string    `orm:"size(30);default(潜在客户)" json:"stage"`
	Phone          string    `orm:"size(30);null" json:"phone"`
	Mobile         string    `orm:"size(30);null" json:"mobile"`
	Website        string    `orm:"size(200);null" json:"website"`
	Source         string    `orm:"size(60);null" json:"source"`
	NextContact    time.Time `orm:"type(datetime);null" json:"next_contact"`
	Note           string    `orm:"type(text);null" json:"note"`
	Creator        string    `orm:"size(50);null" json:"creator"`
	Owner          string    `orm:"size(50);null" json:"owner"`
	FollowUpRecord string    `orm:"type(text);null" json:"follow_up_record"`
	Province       string    `orm:"size(40);null" json:"province"`
	City           string    `orm:"size(40);null" json:"city"`
	District       string    `orm:"size(40);null" json:"district"`
	Address        string    `orm:"size(200);null" json:"address"`
	Email          string    `orm:"size(120);null" json:"email"`
	Amount         float64   `orm:"digits(12);decimals(2);default(0)" json:"amount"`
	CreatedAt      time.Time `orm:"auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt      time.Time `orm:"auto_now;type(datetime)" json:"updated_at"`
}

type Contact struct {
	Id        int       `orm:"auto" json:"id"`
	Customer  *Customer `orm:"rel(fk);on_delete(cascade)" json:"customer,omitempty"`
	Name      string    `orm:"size(60)" json:"name"`
	Role      string    `orm:"size(60);null" json:"role"`
	Phone     string    `orm:"size(30);null" json:"phone"`
	Email     string    `orm:"size(120);null" json:"email"`
	CreatedAt time.Time `orm:"auto_now_add;type(datetime)" json:"created_at"`
}

type Activity struct {
	Id         int       `orm:"auto" json:"id"`
	Customer   *Customer `orm:"rel(fk);on_delete(cascade)" json:"customer,omitempty"`
	Type       string    `orm:"size(30)" json:"type"`
	Content    string    `orm:"type(text)" json:"content"`
	NextAction string    `orm:"size(200);null" json:"next_action"`
	CreatedAt  time.Time `orm:"auto_now_add;type(datetime)" json:"created_at"`
}

type Contract struct {
	Id          int       `orm:"auto" json:"id"`
	Customer    *Customer `orm:"rel(fk);on_delete(cascade)" json:"customer,omitempty"`
	SerialNo    string    `orm:"size(80);null" json:"serial_no"`
	Title       string    `orm:"size(120)" json:"title"`
	BusinessName string   `orm:"size(120);null" json:"business_name"`
	Amount      float64   `orm:"digits(12);decimals(2);default(0)" json:"amount"`
	OrderDate   time.Time `orm:"type(datetime);null" json:"order_date"`
	StartDate   time.Time `orm:"type(datetime);null" json:"start_date"`
	EndDate     time.Time `orm:"type(datetime);null" json:"end_date"`
	CustomerSigner string `orm:"size(80);null" json:"customer_signer"`
	CompanySigner  string `orm:"size(80);null" json:"company_signer"`
	Content     string    `orm:"type(text);null" json:"content"`
	Attachments string    `orm:"type(text);null" json:"attachments"`
	Products    string    `orm:"type(text);null" json:"products"`
	Status      string    `orm:"size(20);default(pending)" json:"status"`
	Submitter   string    `orm:"size(50)" json:"submitter"`
	Reviewer    string    `orm:"size(50);null" json:"reviewer"`
	ReviewNote  string    `orm:"type(text);null" json:"review_note"`
	CreatedAt   time.Time `orm:"auto_now_add;type(datetime)" json:"created_at"`
	ReviewedAt  time.Time `orm:"type(datetime);null" json:"reviewed_at"`
}

type Payment struct {
	Id             int       `orm:"auto" json:"id"`
	Customer       *Customer `orm:"rel(fk);on_delete(cascade)" json:"customer,omitempty"`
	Contract       *Contract `orm:"rel(fk);on_delete(cascade)" json:"contract,omitempty"`
	SerialNo       string    `orm:"size(80);null" json:"serial_no"`
	ContractTitle  string    `orm:"size(120);null" json:"contract_title"`
	ContractAmount float64   `orm:"digits(12);decimals(2);default(0)" json:"contract_amount"`
	PaymentMethod  string    `orm:"size(40);null" json:"payment_method"`
	PaymentAmount  float64   `orm:"digits(12);decimals(2);default(0)" json:"payment_amount"`
	Status         string    `orm:"size(20);default(pending)" json:"status"`
	PaymentDate    time.Time `orm:"type(datetime);null" json:"payment_date"`
	Note           string    `orm:"type(text);null" json:"note"`
	CreatedAt      time.Time `orm:"auto_now_add;type(datetime)" json:"created_at"`
}

type User struct {
	Id        int       `orm:"auto" json:"id"`
	Username  string    `orm:"size(50)" json:"username"`
	Alias     string    `orm:"size(50);null" json:"alias"`
	Password  string    `orm:"size(120)" json:"-"`
	Role      string    `orm:"size(20);default(user)" json:"role"`
	CreatedAt time.Time `orm:"auto_now_add;type(datetime)" json:"created_at"`
}

func init() {
	orm.RegisterModel(new(Customer), new(Contact), new(Activity), new(Contract), new(Payment), new(User))
}

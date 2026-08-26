package controllers

import "github.com/beego/beego/v2/server/web"

func Register() {
	web.Router("/api/auth/login", &AuthController{}, "post:Login")
	web.Router("/api/auth/change-password", &AuthController{}, "post:ChangePassword")
	web.Router("/api/app-config", &ConfigController{}, "get:Get")
	web.Router("/api/dashboard/summary", &DashboardController{}, "get:Summary")
	web.Router("/api/customers/import", &CustomerController{}, "post:Import")
	web.Router("/api/customers/export", &CustomerController{}, "get:Export")
	web.Router("/api/customers", &CustomerController{}, "get:List;post:Create")
	web.Router("/api/customers/:id/to-pool", &CustomerController{}, "post:TransferToPool")
	web.Router("/api/customers/:id/claim", &CustomerController{}, "post:Claim")
	web.Router("/api/customers/:id", &CustomerController{}, "put:Update;delete:Delete")
	web.Router("/api/contracts", &ContractController{}, "get:List;post:Create")
	web.Router("/api/contracts/:id/review", &ContractController{}, "post:Review")
	web.Router("/api/users", &UserController{}, "get:List;post:Create")
	web.Router("/api/users/:id", &UserController{}, "put:Update;delete:Delete")
	web.Router("/api/contacts", &ContactController{}, "get:List;post:Create")
	web.Router("/api/activities", &ActivityController{}, "get:List;post:Create")
	web.Router("/api/payments", &PaymentController{}, "get:List;post:Create")
	web.Router("/api/payments/:id/review", &PaymentController{}, "post:Review")
}

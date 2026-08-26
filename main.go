package main

import (
	"fmt"
	"os"
	"strings"

	"crm/controllers"
	"crm/models"
	"crm/security"
	_ "crm/routers"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

func main() {
	if handled, err := handleWindowsServiceCommand(os.Args[1:]); handled {
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		return
	}
	if shouldRunAsWindowsService() {
		if err := runWindowsService(); err != nil {
			fmt.Printf("windows service failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	runApp()
}

func runApp() {
	driver := web.AppConfig.DefaultString("db_driver", "mysql")
	ormDriver := driver
	user := web.AppConfig.DefaultString("db_user", "root")
	password := web.AppConfig.DefaultString("db_password", "123456")
	host := web.AppConfig.DefaultString("db_host", "127.0.0.1:3306")
	name := web.AppConfig.DefaultString("db_name", "crm")
	sqlitePath := web.AppConfig.DefaultString("db_path", "crm.db")

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&loc=Local", user, password, host, name)
	if driver == "sqlite3" {
		ormDriver = "sqlite"
		dsn = "file:" + sqlitePath + "?cache=shared&mode=rwc"
	}
	if err := orm.RegisterDataBase("default", ormDriver, dsn); err != nil {
		fmt.Printf("database connection failed: %v\n", err)
		fmt.Println("please create the crm database and check CRM_DB_* environment variables")
		os.Exit(1)
	}
	orm.RegisterModel(new(models.Customer), new(models.Contact), new(models.Activity), new(models.Contract), new(models.Payment), new(models.User))
	if err := orm.RunSyncdb("default", false, true); err != nil {
		fmt.Printf("database migration failed: %v\n", err)
		os.Exit(1)
	}
	ensureContractSchema()
	ensureUserSchema()
	migrateUserPasswords()
	ensureAdminUser()
	web.SetStaticPath("/", "static")
	controllers.Register()
	web.Run()
}

func ensureAdminUser() {
	var users []models.User
	_, err := orm.NewOrm().QueryTable(new(models.User)).Filter("username", "admin").All(&users)
	if err == nil && len(users) == 0 {
		hashed, _ := security.HashPassword("admin")
		_, _ = orm.NewOrm().Insert(&models.User{
			Username: "admin",
			Alias:    "管理员",
			Password: hashed,
			Role:     "admin",
		})
	}
}

func migrateUserPasswords() {
	var users []models.User
	_, err := orm.NewOrm().QueryTable(new(models.User)).All(&users)
	if err != nil {
		return
	}
	for _, user := range users {
		if security.IsPasswordHashed(user.Password) {
			continue
		}
		hashed, err := security.HashPassword(user.Password)
		if err != nil {
			continue
		}
		user.Password = hashed
		_, _ = orm.NewOrm().Update(&user, "Password")
	}
}

func ensureUserSchema() {
	if web.AppConfig.DefaultString("db_driver", "mysql") == "sqlite3" {
		return
	}
	if err := orm.Exec("ALTER TABLE user ADD COLUMN alias VARCHAR(50) NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("user schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE user ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'user'"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("user schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE user ADD COLUMN created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("user schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
}

func ensureContractSchema() {
	if web.AppConfig.DefaultString("db_driver", "mysql") == "sqlite3" {
		return
	}
	if err := orm.Exec("ALTER TABLE contract ADD COLUMN serial_no VARCHAR(80) NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("contract schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE contract ADD COLUMN business_name VARCHAR(120) NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("contract schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE contract ADD COLUMN order_date DATETIME NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("contract schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE contract ADD COLUMN start_date DATETIME NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("contract schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE contract ADD COLUMN end_date DATETIME NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("contract schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE contract ADD COLUMN customer_signer VARCHAR(80) NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("contract schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE contract ADD COLUMN company_signer VARCHAR(80) NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("contract schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE contract ADD COLUMN attachments LONGTEXT NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("contract schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE contract ADD COLUMN products LONGTEXT NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			fmt.Printf("contract schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE payment ADD COLUMN serial_no VARCHAR(80) NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") && !strings.Contains(strings.ToLower(err.Error()), "check that column/key exists") && !strings.Contains(strings.ToLower(err.Error()), "doesn't exist") {
			fmt.Printf("payment schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE payment ADD COLUMN payment_method VARCHAR(40) NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") && !strings.Contains(strings.ToLower(err.Error()), "check that column/key exists") && !strings.Contains(strings.ToLower(err.Error()), "doesn't exist") {
			fmt.Printf("payment schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE payment ADD COLUMN note LONGTEXT NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") && !strings.Contains(strings.ToLower(err.Error()), "check that column/key exists") && !strings.Contains(strings.ToLower(err.Error()), "doesn't exist") {
			fmt.Printf("payment schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE payment ADD COLUMN contract_title VARCHAR(120) NULL"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") && !strings.Contains(strings.ToLower(err.Error()), "check that column/key exists") && !strings.Contains(strings.ToLower(err.Error()), "doesn't exist") {
			fmt.Printf("payment schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
	if err := orm.Exec("ALTER TABLE payment ADD COLUMN contract_amount DECIMAL(12,2) NOT NULL DEFAULT 0"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") && !strings.Contains(strings.ToLower(err.Error()), "check that column/key exists") && !strings.Contains(strings.ToLower(err.Error()), "doesn't exist") {
			fmt.Printf("payment schema migration failed: %v\n", err)
			os.Exit(1)
		}
	}
}

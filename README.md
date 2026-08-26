# Northstar CRM

一个基于 `Beego v2 + MySQL + 原生 HTML/CSS/JS` 的轻量 CRM。

## 功能说明
传统行业的福音，致力于打造一个完全开源的生态，真正的免费CRM，完全开源，go编写搭建简单，支持Windows、Linux、mac
同时支持mysql和sqlite双数据库，根据自己的需求进行选择，品牌名称可以自己配置，
如果你有需求不满足联系QQ982242224 即可添加

### 客户管理

- 新建客户
- 客户列表分页
- 成交状态筛选
- 客户名称点击查看详情
- 客户名称悬浮显示电话
- CSV 导入客户
- CSV 导出客户
- 客户详情页内编辑基础信息
- 客户转移到公海池

### 客户详情

- 跟进记录
- 基本信息
- 联系人
- 合同
- 回款信息
- 新增跟进记录
- 新增联系人
- 新增合同
- 新增回款

### 跟进记录

- 支持手动新增
- 导入客户时可同步写入跟进记录表
- 待办事项按当天需要联系的客户自动统计
- 已转移到公海池的客户不进入待办

### 公海池

- 公海池客户列表
- 公海池客户领用
- 公海池展示字段与客户管理一致

### 合同管理

- 新建合同
- 合同列表
- 合同详情弹框
- 产品明细
- 附件上传
- 备注
- 合同审批

### 回款管理

- 新建回款
- 回款列表
- 回款审批

### 账户管理

- 管理员创建账户
- 管理员编辑账户别名和密码
- 管理员删除普通账户
- 普通用户修改自己的密码

### 权限控制

- `admin` 可查看全部客户、合同、回款、账户
- 普通账户只能查看自己相关数据
- 公海池客户可被普通账户领用

### 密码安全

- 用户密码不再明文保存
- 使用不可逆哈希存储
- 新建/修改密码时会做强度校验

### 首页总览

首页展示以下指标：

- 本月成交金额
- 本年度成交金额
- 本月成交单数
- 本年成交单数
- 本月新增联系人
- 本年新增联系人
- 总成交
- 总联系人
- 总成交金额
- 本月回款金额
- 本年回款金额
- 总回款金额
- 本月待回款金额
- 本年待回款金额
- 总待回款金额

管理员显示全部数据，普通用户显示当前账号相关数据。

## 客户字段

当前客户维度统一为：

- 客户名称
- 客户级别
- 客户行业
- 客户来源
- 成交状态
- 电话
- 网址
- 下次联系时间
- 备注
- 创建人
- 更新时间
- 创建时间
- 负责人
- 跟进记录
- 省
- 市
- 区/县
- 详细地址

## 导入导出

### 导入

CSV 支持字段：

1. 客户名称
2. 客户级别
3. 客户行业
4. 客户来源
5. 成交状态
6. 电话
7. 网址
8. 下次联系时间
9. 备注
10. 手机
11. 创建人
12. 更新时间
13. 创建时间
14. 负责人
15. 跟进记录
16. 省
17. 市
18. 区/县
19. 详细地址

导入规则：

- 支持 CSV 表头变化
- 导入后“创建人”和“负责人”会改成当前导入账户
- 跟进记录会同步写入跟进记录表
- 导入过程会显示进度条
- 导入失败会显示失败条目和原因

### 导出

- 导出格式为 CSV
- 导出字段顺序与导入模板一致

## 默认管理员

- 用户名：`admin`
- 密码：`admin`

## 配置说明

默认读取：

```text
conf/app.conf
```

常用配置：

```ini
appname = crm
brand_name = 产品名称
workspace_name = 销售工作台
httpport = 9080
runmode = dev
db_driver = mysql
db_user = root
db_password = 123456
db_host = 127.0.0.1:3306
db_name = crm
db_path = crm.db
uploads = uploads
login_user = admin
login_password = admin
```

SQLite 示例：

```ini
db_driver = sqlite3
db_path = crm.db
```

## 启动方式

### 普通启动

```powershell
go run .
```

或：

```powershell
go build -o crm.exe
.\crm.exe
```

### Windows 服务

```powershell
.\crm.exe --service install
.\crm.exe --service start
.\crm.exe --service stop
.\crm.exe --service restart
.\crm.exe --service status
.\crm.exe --service uninstall
```

说明：

- `install` 安装为 Windows 服务
- 服务默认自动启动
- 服务模式会自动切换到程序所在目录

### Linux 编译

```powershell
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o crm-linux
```

### Linux 部署建议

- 把程序、`conf/`、`static/`、`uploads/` 放在同一目录
- 使用 `systemd` 守护进程
- 推荐配合 Nginx 反向代理

## 接口说明

### 认证

- `POST /api/auth/login`
- `POST /api/auth/change-password`

### 客户

- `GET /api/customers`
- `POST /api/customers`
- `PUT /api/customers/:id`
- `DELETE /api/customers/:id`
- `POST /api/customers/import`
- `GET /api/customers/export`
- `POST /api/customers/:id/to-pool`
- `POST /api/customers/:id/claim`

### 跟进记录

- `GET /api/activities`
- `POST /api/activities`

### 联系人

- `GET /api/contacts`
- `POST /api/contacts`

### 合同

- `GET /api/contracts`
- `POST /api/contracts`
- `POST /api/contracts/:id/review`

### 回款

- `GET /api/payments`
- `POST /api/payments`
- `POST /api/payments/:id/review`

### 账户

- `GET /api/users`
- `POST /api/users`
- `PUT /api/users/:id`
- `DELETE /api/users/:id`

## 说明

- 数据库表会在启动时自动创建
- 历史旧表缺少字段时会自动补齐部分字段
- 静态页面为原生页面，便于后续继续扩展

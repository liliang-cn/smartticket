# SmartTicket 部门 / 组织层级设计(Departments）

> 日期:2026-06-12 · Program 第 4 份子 spec(由「升级要找上级」需求衍生)。横切影响 spec A(Sentinel 升级)与现有 `EscalateAutomation`。
> 决策:部门与团队**正交**(Team 管路由,Department 管汇报) · 一人归属一个部门 · 部门经理**可见子树下属工单**(选项 B,扩 RBAC) · 非破坏性靠 `DepartmentIsolation` 开关(默认关)。

## 目标与非目标

**目标**:引入真实的组织汇报层级,让

1. **"找上级"成立** —— 升级能路由到具体的人(部门经理),而不只是涨优先级。
2. **部门经理可见下属工单** —— 经理能看本部门(含子部门)成员经手的工单。
3. 为后续**按部门报表/分析**打底。

**非目标(YAGNI)**:一人多部门(汇报维度 1:1)、矩阵式双线汇报、部门级独立权限模板(沿用现有 Role)、跨部门审批流。

## 既有架构约定 / 现状

- 可见性:`internal/ticket/service.go` 的 `scopeToActor(q, actor)` —— 客户 actor 限 `customer_id`;**staff/admin 当前看全部,无限制**。
- 升级:`EscalateAutomation`(`ticket/service.go:1345`)仅涨优先级一档,不通知人。
- Team 平铺 `{Name,Description}` + `TeamMember{TeamID,UserID}` 关联;工单挂 `AssignedTo`(人)+ `AssignedTeamID`(组),二者保留不动。
- `authz.Actor{UserID, Role, CustomerID}` 由 `ActorFromUser(u)` 构建。
- 模型集中 `models.go`,Unix 时间戳;端口 6533;后台 `web/src/pages/`,7 语言 i18n。

---

## 数据模型 — `internal/models/models.go`

```go
type Department struct {
    BaseModel
    Name      string      `gorm:"size:120;not null"`
    ParentID  *uint       `gorm:"index"`        // 嵌套树,nil = 顶级
    Parent    *Department `gorm:"foreignKey:ParentID"`
    ManagerID *uint       `gorm:"index"`        // 部门负责人(一个 User)
    Manager   *User       `gorm:"foreignKey:ManagerID"`
}

// User 增加:
//   DepartmentID *uint `gorm:"index"`   // 归属部门(汇报维度,可空)
```

约束:`ParentID` 不能成环(建/改时校验);`ManagerID` 指向存在且为 staff 的用户。

---

## "找上级" — supervisorOf

`internal/department/service.go`:

```
supervisorOf(user) User?:
  dept := user.Department
  若 dept == nil → nil
  若 dept.Manager 存在 且 dept.ManagerID ≠ user.ID → 返回 dept.Manager
  否则(user 本人即经理,或本部门无经理)→ 沿 ParentID 上溯到父部门,返回其 Manager
  到树顶仍无 → nil(组织最高层,无上级)
```

部门树深度有限,递归上溯;加内存缓存(部门树变更时失效)避免逐次查库。

---

## 接到升级(改 `EscalateAutomation` + spec A Sentinel)

```
工单升级:
  取 ticket.AssignedTo
  sup := supervisorOf(assignee)
  若 sup 存在 → 通知 sup(站内 notification + 可选邮件)+ 涨优先级一档
  若 assignee 为空 或 sup 为空 → 退回纯涨优先级(现有行为,不回退体验)
```

- spec A 的 **Sentinel** 输出 `escalate_to: manager` → 解析为 `supervisorOf(assignee)`;Copilot 卡的「升级」采纳动作走此路径。
- 通知复用现有 `internal/notification`。

---

## RBAC 扩展:部门经理可见子树(选项 B)

### Actor 加部门作用域

```go
// authz.Actor 增加
DeptScope []uint   // 可见部门 ID 集 = 本人所辖部门 + 所有后代部门
```

构建时机:`ActorFromUser` / 认证中间件 —— 若用户是任一部门的 `ManagerID`,计算该部门子树的所有部门 ID 填入 `DeptScope`(普通成员 `DeptScope` 为空)。

### scopeToActor 扩展 — `internal/ticket/service.go`

```
客户 actor      → 现有 customer_id 逻辑,不变
admin           → 全部,不变
其余 staff:
  若 DepartmentIsolation 关(默认):
      普通坐席 → 全部(现有行为,不破坏)
      部门经理 → 默认全部;其「我的部门」视图/升级收件箱用 DeptScope 过滤:
                 assigned_to IN (SELECT id FROM users WHERE department_id IN DeptScope)
  若 DepartmentIsolation 开:
      部门经理 → 限 DeptScope 子树(同上过滤)
      普通成员 → 限本人 department_id
```

`DepartmentIsolation` 为 `SystemSetting`/`AISettings` 同类的全局开关(默认 **false**),让默认行为零变化,需要强隔离的组织再开。

### API

```
GET /api/v1/tickets?scope=my_department   部门经理:返回 DeptScope 过滤后的工单(管理视图)
```
不带 `scope` 时维持现有行为(受 DepartmentIsolation 控制)。

---

## 后台 UI

- `web/src/pages/departments.tsx`:部门**树编辑器**(建/嵌套/拖拽排序/删)+ 每部门设 Manager + 挂成员。
- 用户编辑页加「部门」下拉。
- 工单列表加「我的部门」过滤(仅对部门经理显示)。
- Settings 加 `DepartmentIsolation` 开关。7 语言 i18n。

---

## 测试策略

- 模型:`ParentID` 成环被拒;`ManagerID` 非 staff 被拒。
- `supervisorOf`:经理是本人时上溯父部门;多级上溯;到顶返回 nil;无部门返回 nil。
- 升级:assignee 有上级→通知+涨优先级;无 assignee / 无上级→仅涨优先级(回归现有)。
- RBAC:`DeptScope` 计算覆盖子树;隔离关时普通坐席看全部、经理 `my_department` 受限;隔离开时成员限本部门;admin 恒看全部;客户隔离不受影响(回归)。

## 实现顺序(spec D 内部)

1. `Department` 模型 + `User.DepartmentID` + `AutoMigrate` + 成环/Manager 校验。
2. `internal/department`:CRUD + `supervisorOf`(带缓存)+ `DeptScope` 计算。
3. 改 `EscalateAutomation` 接 `supervisorOf` + 通知;spec A Sentinel 升级动作接同一路径。
4. `Actor.DeptScope` 填充 + `scopeToActor` 扩展 + `?scope=my_department` + `DepartmentIsolation` 开关。
5. 后台部门树编辑器 + 用户部门字段 + 工单「我的部门」过滤 + i18n。
6. `swag init --parseDependency` 刷新 OpenAPI。
